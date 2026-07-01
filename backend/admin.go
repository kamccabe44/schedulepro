package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type BarberUser struct {
	UserID string `json:"userId"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// adminAppointments returns all appointments for a given date.
// Accessible to barbers and admins.
func adminAppointments(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !isBarberOrAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	date := req.QueryStringParameters["date"]
	if date == "" {
		return respond(400, map[string]string{"error": "date query parameter required (YYYY-MM-DD)"})
	}

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              aws.String("date-index"),
		KeyConditionExpression: aws.String("#date = :date"),
		ExpressionAttributeNames: map[string]string{
			"#date": "date",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":date": &types.AttributeValueMemberS{Value: date},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	var appts []Appointment
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &appts); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if appts == nil {
		appts = []Appointment{}
	}
	return respond(200, appts)
}

// listBarbers returns all users in the barbers group.
// Accessible to admins only.
func listBarbers(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !isAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	out, err := cognitoClient.ListUsersInGroup(ctx, &cognitoidentityprovider.ListUsersInGroupInput{
		UserPoolId: &userPoolID,
		GroupName:  aws.String("barbers"),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	barbers := make([]BarberUser, 0, len(out.Users))
	for _, u := range out.Users {
		b := BarberUser{UserID: aws.ToString(u.Username)}
		for _, attr := range u.Attributes {
			switch aws.ToString(attr.Name) {
			case "email":
				b.Email = aws.ToString(attr.Value)
			case "name":
				b.Name = aws.ToString(attr.Value)
			}
		}
		barbers = append(barbers, b)
	}
	return respond(200, barbers)
}

// addBarber finds a user by email and adds them to the barbers group.
// Accessible to admins only.
func addBarber(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !isAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil || body.Email == "" {
		return respond(400, map[string]string{"error": "email is required"})
	}

	// Find the user by email
	users, err := cognitoClient.ListUsers(ctx, &cognitoidentityprovider.ListUsersInput{
		UserPoolId: &userPoolID,
		Filter:     aws.String(`email = "` + body.Email + `"`),
		Limit:      aws.Int32(1),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if len(users.Users) == 0 {
		return respond(404, map[string]string{"error": "no account found with that email address"})
	}

	username := aws.ToString(users.Users[0].Username)

	_, err = cognitoClient.AdminAddUserToGroup(ctx, &cognitoidentityprovider.AdminAddUserToGroupInput{
		UserPoolId: &userPoolID,
		Username:   &username,
		GroupName:  aws.String("barbers"),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(200, map[string]string{"message": "barber added successfully"})
}

// removeBarber removes a user from the barbers group by their Cognito username (sub).
// Accessible to admins only.
func removeBarber(ctx context.Context, req events.APIGatewayV2HTTPRequest, userID string) (events.APIGatewayV2HTTPResponse, error) {
	if !isAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	_, err := cognitoClient.AdminRemoveUserFromGroup(ctx, &cognitoidentityprovider.AdminRemoveUserFromGroupInput{
		UserPoolId: &userPoolID,
		Username:   &userID,
		GroupName:  aws.String("barbers"),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(200, map[string]string{"message": "barber removed"})
}

func listBarbersPublic(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	out, err := cognitoClient.ListUsersInGroup(ctx, &cognitoidentityprovider.ListUsersInGroupInput{
		UserPoolId: &userPoolID,
		GroupName:  aws.String("barbers"),
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	barbers := make([]BarberUser, 0, len(out.Users))
	for _, u := range out.Users {
		b := BarberUser{UserID: aws.ToString(u.Username)}
		for _, attr := range u.Attributes {
			switch aws.ToString(attr.Name) {
			case "email":
				b.Email = aws.ToString(attr.Value)
			case "name":
				b.Name = aws.ToString(attr.Value)
			}
		}
		barbers = append(barbers, b)
	}
	return respond(200, barbers)
}
