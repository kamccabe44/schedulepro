package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

// listSlots returns available slots for a date and barber.
// Requires barberId query param. Public — no auth required.
func listSlots(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	date := req.QueryStringParameters["date"]
	barberID := req.QueryStringParameters["barberId"]
	if date == "" || barberID == "" {
		return respond(400, map[string]string{"error": "date and barberId query parameters required"})
	}

	// Fetch barber's schedule
	settingsResult, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &barberSettingsTable,
		Key: map[string]types.AttributeValue{
			"barberId": &types.AttributeValueMemberS{Value: barberID},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if settingsResult.Item == nil {
		return respond(200, []SlotResponse{})
	}

	var settings BarberSettings
	if err := attributevalue.UnmarshalMap(settingsResult.Item, &settings); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		return respond(400, map[string]string{"error": "invalid date format, expected YYYY-MM-DD"})
	}

	daySchedule, ok := settings.Schedule[d.Weekday().String()]
	if !ok || !daySchedule.Open {
		return respond(200, []SlotResponse{})
	}

	allSlots, err := generateSlotsFromSchedule(date, daySchedule)
	if err != nil {
		return respond(400, map[string]string{"error": err.Error()})
	}

	// Find which of this barber's slots are already booked
	out, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              aws.String("date-index"),
		KeyConditionExpression: aws.String("#date = :date"),
		FilterExpression:       aws.String("barberId = :bid AND #status = :booked"),
		ExpressionAttributeNames: map[string]string{
			"#date":   "date",
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":date":   &types.AttributeValueMemberS{Value: date},
			":bid":    &types.AttributeValueMemberS{Value: barberID},
			":booked": &types.AttributeValueMemberS{Value: "booked"},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	booked := map[string]bool{}
	for _, item := range out.Items {
		if ts, ok := item["timeSlot"].(*types.AttributeValueMemberS); ok {
			booked[ts.Value] = true
		}
	}

	result := make([]SlotResponse, len(allSlots))
	for i, ts := range allSlots {
		result[i] = SlotResponse{Date: date, TimeSlot: ts, Available: !booked[ts]}
	}
	return respond(200, result)
}

// listServices returns the available service options.
func listServices() (events.APIGatewayV2HTTPResponse, error) {
	list := make([]Service, 0, len(services))
	for _, s := range services {
		list = append(list, s)
	}
	return respond(200, list)
}

// bookAppointment creates a new appointment for the authenticated user.
func bookAppointment(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, userEmail, userName := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}

	var body struct {
		Date       string `json:"date"`
		TimeSlot   string `json:"timeSlot"`
		Service    string `json:"service"`
		Notes      string `json:"notes"`
		BarberID   string `json:"barberId"`
		BarberName string `json:"barberName"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return respond(400, map[string]string{"error": "invalid request body"})
	}

	if body.Date == "" || body.TimeSlot == "" || body.Service == "" || body.BarberID == "" {
		return respond(400, map[string]string{"error": "date, timeSlot, service, and barberId are required"})
	}

	// Fetch barber settings to validate service and time slot
	settingsResult, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &barberSettingsTable,
		Key: map[string]types.AttributeValue{
			"barberId": &types.AttributeValueMemberS{Value: body.BarberID},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if settingsResult.Item == nil {
		return respond(400, map[string]string{"error": "barber has not configured their schedule"})
	}
	var barberSettings BarberSettings
	if err := attributevalue.UnmarshalMap(settingsResult.Item, &barberSettings); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	// Validate the service is offered by this barber
	validService := false
	for _, svc := range barberSettings.Services {
		if svc.ID == body.Service {
			validService = true
			break
		}
	}
	if !validService {
		return respond(400, map[string]string{"error": "invalid service for this barber"})
	}

	// Validate the time slot is within the barber's schedule for this day
	d, err := time.Parse("2006-01-02", body.Date)
	if err != nil {
		return respond(400, map[string]string{"error": "invalid date format"})
	}
	daySchedule, ok := barberSettings.Schedule[d.Weekday().String()]
	if !ok || !daySchedule.Open {
		return respond(400, map[string]string{"error": "barber is not available on this day"})
	}
	allSlots, err := generateSlotsFromSchedule(body.Date, daySchedule)
	if err != nil {
		return respond(400, map[string]string{"error": err.Error()})
	}
	validSlot := false
	for _, s := range allSlots {
		if s == body.TimeSlot {
			validSlot = true
			break
		}
	}
	if !validSlot {
		return respond(400, map[string]string{"error": "invalid time slot for this barber"})
	}

	// Check the slot isn't already taken by this barber
	out, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              aws.String("date-index"),
		KeyConditionExpression: aws.String("#date = :date"),
		FilterExpression:       aws.String("barberId = :bid AND timeSlot = :ts AND #status = :booked"),
		ExpressionAttributeNames: map[string]string{
			"#date":   "date",
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":date":   &types.AttributeValueMemberS{Value: body.Date},
			":bid":    &types.AttributeValueMemberS{Value: body.BarberID},
			":ts":     &types.AttributeValueMemberS{Value: body.TimeSlot},
			":booked": &types.AttributeValueMemberS{Value: "booked"},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if out.Count > 0 {
		return respond(409, map[string]string{"error": "this time slot is already booked"})
	}

	appt := Appointment{
		ID:         uuid.New().String(),
		UserID:     userID,
		UserEmail:  userEmail,
		UserName:   userName,
		Date:       body.Date,
		TimeSlot:   body.TimeSlot,
		Service:    body.Service,
		Status:     "booked",
		Notes:      body.Notes,
		BarberID:   body.BarberID,
		BarberName: body.BarberName,
		CreatedAt:  time.Now().UTC(),
	}

	item, err := attributevalue.MarshalMap(appt)
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	_, err = db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	// Send synchronously (not backgrounded) — Lambda freezes the execution
	// environment as soon as the handler returns, which would abandon a goroutine.
	sendBookingConfirmation(ctx, appt)
	sendBarberBookingNotice(ctx, appt)

	return respond(201, appt)
}

// myAppointments returns all appointments for the authenticated user.
func myAppointments(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	userID, _, _ := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}

	// Return appointments from today onward
	today := time.Now().Format("2006-01-02")

	out, err := db.Query(ctx, &dynamodb.QueryInput{
		TableName:              &tableName,
		IndexName:              aws.String("userId-date-index"),
		KeyConditionExpression: aws.String("userId = :uid AND #date >= :today"),
		ExpressionAttributeNames: map[string]string{
			"#date": "date",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid":   &types.AttributeValueMemberS{Value: userID},
			":today": &types.AttributeValueMemberS{Value: today},
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

// cancelAppointment marks the appointment as cancelled.
// Users can only cancel their own appointments.
func cancelAppointment(ctx context.Context, req events.APIGatewayV2HTTPRequest, id string) (events.APIGatewayV2HTTPResponse, error) {
	userID, _, _ := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}

	// Fetch the appointment first to verify ownership
	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if result.Item == nil {
		return respond(404, map[string]string{"error": "appointment not found"})
	}

	var appt Appointment
	if err := attributevalue.UnmarshalMap(result.Item, &appt); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	if appt.UserID != userID {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	if appt.Status == "cancelled" {
		return respond(400, map[string]string{"error": "appointment is already cancelled"})
	}

	_, err = db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :cancelled"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":cancelled": &types.AttributeValueMemberS{Value: "cancelled"},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return respond(404, map[string]string{"error": "appointment not found"})
		}
		return respond(500, map[string]string{"error": err.Error()})
	}

	appt.Status = "cancelled"
	return respond(200, appt)
}

// completeAppointment marks the appointment as completed.
// Only the assigned barber (or an admin) can mark it complete.
func completeAppointment(ctx context.Context, req events.APIGatewayV2HTTPRequest, id string) (events.APIGatewayV2HTTPResponse, error) {
	userID, _, _ := claimsFromRequest(req)
	if userID == "" {
		return respond(401, map[string]string{"error": "unauthorized"})
	}
	if !isBarberOrAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}
	if result.Item == nil {
		return respond(404, map[string]string{"error": "appointment not found"})
	}

	var appt Appointment
	if err := attributevalue.UnmarshalMap(result.Item, &appt); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	if appt.BarberID != userID && !isAdmin(req) {
		return respond(403, map[string]string{"error": "forbidden"})
	}

	if appt.Status == "cancelled" {
		return respond(400, map[string]string{"error": "cannot complete a cancelled appointment"})
	}
	if appt.Status == "completed" {
		return respond(400, map[string]string{"error": "appointment is already completed"})
	}

	_, err = db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression: aws.String("SET #status = :completed"),
		ExpressionAttributeNames: map[string]string{
			"#status": "status",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":completed": &types.AttributeValueMemberS{Value: "completed"},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return respond(404, map[string]string{"error": "appointment not found"})
		}
		return respond(500, map[string]string{"error": err.Error()})
	}

	appt.Status = "completed"
	return respond(200, appt)
}

// claimsFromRequest extracts verified JWT claims injected by API Gateway.
func claimsFromRequest(req events.APIGatewayV2HTTPRequest) (userID, email, name string) {
	claims := req.RequestContext.Authorizer.JWT.Claims
	userID = claims["sub"]
	email = claims["email"]
	name = claims["name"]
	if name == "" {
		name = strings.Split(email, "@")[0]
	}
	return
}

// userGroups returns the Cognito groups the caller belongs to.
// API Gateway serialises the cognito:groups array claim as a JSON array string.
func userGroups(req events.APIGatewayV2HTTPRequest) map[string]bool {
	raw := req.RequestContext.Authorizer.JWT.Claims["cognito:groups"]
	groups := map[string]bool{}
	var list []string
	if err := json.Unmarshal([]byte(raw), &list); err == nil {
		for _, g := range list {
			groups[g] = true
		}
	} else {
		// API Gateway serialises Cognito group arrays as [val1 val2] (no quotes).
		// Strip the surrounding brackets, then split on whitespace or commas.
		raw = strings.Trim(raw, "[]")
		for _, g := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' }) {
			if g = strings.TrimSpace(g); g != "" {
				groups[g] = true
			}
		}
	}
	return groups
}

func isBarberOrAdmin(req events.APIGatewayV2HTTPRequest) bool {
	g := userGroups(req)
	return g["barbers"] || g["admins"]
}

func isAdmin(req events.APIGatewayV2HTTPRequest) bool {
	return userGroups(req)["admins"]
}
