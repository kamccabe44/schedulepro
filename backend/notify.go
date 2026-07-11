package main

import (
	"context"
	"encoding/json"
	"regexp"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/sns"
)

// DeviceToken links a push token (from APNs/FCM via the mobile app) to a user.
// One user can have several devices, so the table is keyed (userId, token).
type DeviceToken struct {
	UserID      string    `json:"userId"      dynamodbav:"userId"`
	Token       string    `json:"token"       dynamodbav:"token"`
	Platform    string    `json:"platform"    dynamodbav:"platform"` // ios | android
	EndpointArn string    `json:"-"           dynamodbav:"endpointArn"`
	UpdatedAt   time.Time `json:"updatedAt"   dynamodbav:"updatedAt"`
}

// snsEndpointArnRe pulls the existing endpoint ARN out of the error SNS returns
// when a token was already registered, so re-registering is idempotent.
var snsEndpointArnRe = regexp.MustCompile(`arn:aws:sns:[^ ]+`)

// registerDevice stores the caller's push token so the backend can send them
// booking notifications. Auth required (runs under the $default JWT route).
//
//	POST /devices   body: { "token": "...", "platform": "ios"|"android" }
func registerDevice(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, _, _ := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}

	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Token == "" {
		return respond(400, map[string]string{"error": "token required"})
	}
	if body.Platform != "ios" && body.Platform != "android" {
		return respond(400, map[string]string{"error": "platform must be ios or android"})
	}

	// Create (or recover) an SNS platform endpoint for this token. If no platform
	// application is configured yet, we still persist the token so nothing breaks
	// and endpoints can be back-filled once credentials are added.
	endpointArn := createPlatformEndpoint(ctx, body.Platform, body.Token)

	item, err := attributevalue.MarshalMap(DeviceToken{
		UserID:      userID,
		Token:       body.Token,
		Platform:    body.Platform,
		EndpointArn: endpointArn,
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if _, err := db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &deviceTokensTable,
		Item:      item,
	}); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(201, map[string]bool{"registered": true})
}

// unregisterDevice removes a token (e.g. on sign-out or when the OS rotates it).
//
//	DELETE /devices/{token}
func unregisterDevice(ctx context.Context, req events.APIGatewayV2HTTPRequest, token string) (events.APIGatewayV2HTTPResponse, error) {
	userID, _, _ := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}
	if _, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &deviceTokensTable,
		Key: map[string]types.AttributeValue{
			"userId": &types.AttributeValueMemberS{Value: userID},
			"token":  &types.AttributeValueMemberS{Value: token},
		},
	}); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	return respond(200, map[string]bool{"unregistered": true})
}

// createPlatformEndpoint registers a token with the matching SNS platform
// application and returns the endpoint ARN. Returns "" (no error) when push is
// not configured, so registration degrades gracefully.
func createPlatformEndpoint(ctx context.Context, platform, token string) string {
	appArn := platformAppArn(platform)
	if appArn == "" || snsClient == nil {
		return ""
	}
	out, err := snsClient.CreatePlatformEndpoint(ctx, &sns.CreatePlatformEndpointInput{
		PlatformApplicationArn: &appArn,
		Token:                  &token,
	})
	if err != nil {
		// SNS returns the pre-existing ARN in the error text when the token is
		// already registered — recover it so this stays idempotent.
		if arn := snsEndpointArnRe.FindString(err.Error()); arn != "" {
			return arn
		}
		return ""
	}
	return aws.ToString(out.EndpointArn)
}

// sendPushToUser delivers a notification to every device a user has registered.
// Best-effort and synchronous (Lambda freezes after the handler returns) — it
// mirrors how booking emails are sent and never fails the calling request.
func sendPushToUser(ctx context.Context, userID, title, message string) {
	if snsClient == nil || deviceTokensTable == "" {
		return
	}
	out, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &deviceTokensTable,
		KeyConditionExpression: aws.String("userId = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	})
	if err != nil {
		return
	}
	for _, item := range out.Items {
		var d DeviceToken
		if attributevalue.UnmarshalMap(item, &d) != nil || d.EndpointArn == "" {
			continue
		}
		publishToEndpoint(ctx, d, title, message)
	}
}

// publishToEndpoint sends one notification to one device endpoint, formatting
// the payload for the device's platform (APNs vs FCM).
func publishToEndpoint(ctx context.Context, d DeviceToken, title, message string) {
	payload, err := buildPushPayload(d.Platform, title, message)
	if err != nil {
		return
	}
	_, _ = snsClient.Publish(ctx, &sns.PublishInput{
		TargetArn:        &d.EndpointArn,
		Message:          &payload,
		MessageStructure: aws.String("json"),
	})
}

// buildPushPayload produces the SNS "json" message structure with a platform
// specific inner payload. SNS requires each key to be a JSON-encoded string.
func buildPushPayload(platform, title, message string) (string, error) {
	outer := map[string]string{"default": message}

	switch platform {
	case "ios":
		aps, _ := json.Marshal(map[string]any{
			"aps": map[string]any{
				"alert": map[string]string{"title": title, "body": message},
				"sound": "default",
			},
		})
		// Sandbox key targets the APNs sandbox platform app used for TestFlight
		// and development builds; production uses "APNS". We set both so the
		// message is delivered whichever platform app the endpoint belongs to.
		outer["APNS"] = string(aps)
		outer["APNS_SANDBOX"] = string(aps)
	case "android":
		gcm, _ := json.Marshal(map[string]any{
			"notification": map[string]string{"title": title, "body": message},
		})
		outer["GCM"] = string(gcm)
	}

	b, err := json.Marshal(outer)
	return string(b), err
}
