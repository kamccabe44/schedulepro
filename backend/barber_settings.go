package main

import (
	"context"
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// getBarberSettings returns a barber's schedule and services. Public.
func getBarberSettings(ctx context.Context, barberID string) (events.APIGatewayV2HTTPResponse, error) {
	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &barberSettingsTable,
		Key: map[string]types.AttributeValue{
			"barberId": &types.AttributeValueMemberS{Value: barberID},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if result.Item == nil {
		return respond(200, BarberSettings{
			BarberID: barberID,
			Schedule: map[string]DaySchedule{},
			Services: []BarberService{},
		})
	}
	var settings BarberSettings
	if err := attributevalue.UnmarshalMap(result.Item, &settings); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if settings.Services == nil {
		settings.Services = []BarberService{}
	}
	if settings.Schedule == nil {
		settings.Schedule = map[string]DaySchedule{}
	}
	return respond(200, settings)
}

// updateBarberSettings saves the authenticated barber's schedule and services.
func updateBarberSettings(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	if !isBarberOrAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}
	userID, _, _ := claimsFromRequest(req)

	var body BarberSettings
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return respond(400, map[string]string{"error": "invalid request body"})
	}
	body.BarberID = userID

	for _, svc := range body.Services {
		if svc.ID == "" || svc.Name == "" || svc.Duration <= 0 {
			return respond(400, map[string]string{"error": "each service requires id, name, and duration > 0"})
		}
	}

	item, err := attributevalue.MarshalMap(body)
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if _, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &barberSettingsTable,
		Item:      item,
	}); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	return respond(200, body)
}
