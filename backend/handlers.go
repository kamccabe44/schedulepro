package main

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
)

func listEvents(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	input := &dynamodb.ScanInput{
		TableName: &tableName,
	}

	start := req.QueryStringParameters["start"]
	end := req.QueryStringParameters["end"]

	if start != "" && end != "" {
		input.FilterExpression = aws.String("startDate BETWEEN :s AND :e")
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":s": &types.AttributeValueMemberS{Value: start},
			":e": &types.AttributeValueMemberS{Value: end},
		}
	}

	out, err := db.Scan(ctx, input)
	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	var evs []Event
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &evs); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(200, evs)
}

func createEvent(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	var ev Event
	if err := json.Unmarshal([]byte(req.Body), &ev); err != nil {
		return respond(400, map[string]string{"error": "invalid request body"})
	}

	if strings.TrimSpace(ev.Title) == "" || ev.StartTime == "" || ev.EndTime == "" {
		return respond(400, map[string]string{"error": "title, startTime, and endTime are required"})

	}

	ev.ID = uuid.New().String()
	ev.StartDate = ev.StartTime[:10]
	ev.CreatedAt = time.Now().UTC()
	if ev.Color == "" {
		ev.Color = "#3b82f6"
	}

	item, err := attributevalue.MarshalMap(ev)
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

	return respond(201, ev)
}

func getEvent(ctx context.Context, id string) (events.APIGatewayV2HTTPResponse, error) {
	out, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
	})

	if err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	if out.Item == nil {
		return respond(404, map[string]string{"error": "event not found"})
	}

	var ev Event
	if err := attributevalue.UnmarshalMap(out.Item, &ev); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(200, ev)
}

func updateEvent(ctx context.Context, req events.APIGatewayV2HTTPRequest, id string) (events.APIGatewayV2HTTPResponse, error) {
	var body map[string]string
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return respond(400, map[string]string{"error": "invalid request body"})
	}

	allowed := []string{"title", "startTime", "endTime", "description", "location", "color"}

	var setClauses []string
	exprNames := map[string]string{}
	exprValues := map[string]types.AttributeValue{}

	for _, field := range allowed {
		val, ok := body[field]
		if !ok {
			continue
		}
		setClauses = append(setClauses, "#"+field+" = :"+field)
		exprNames["#"+field] = field
		exprValues[":"+field] = &types.AttributeValueMemberS{Value: val}
	}

	if len(setClauses) == 0 {
		return respond(400, map[string]string{"error": "no valid fields to update"})
	}

	if st, ok := body["startTime"]; ok {
		setClauses = append(setClauses, "#startDate = :startDate")
		exprNames["#startDate"] = "startDate"
		exprValues[":startDate"] = &types.AttributeValueMemberS{Value: st[:10]}
	}

	expr := "SET " + strings.Join(setClauses, ", ")

	out, err := db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		UpdateExpression:          aws.String(expr),
		ExpressionAttributeNames:  exprNames,
		ExpressionAttributeValues: exprValues,
		ConditionExpression:       aws.String("attribute_exists(id)"),
		ReturnValues:              types.ReturnValueAllNew,
	})

	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return respond(404, map[string]string{"error": "event not found"})
		}
		return respond(500, map[string]string{"error": err.Error()})
	}

	var ev Event
	if err := attributevalue.UnmarshalMap(out.Attributes, &ev); err != nil {
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(200, ev)
}

func deleteEvent(ctx context.Context, id string) (events.APIGatewayV2HTTPResponse, error) {
	_, err := db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: &tableName,
		Key: map[string]types.AttributeValue{
			"id": &types.AttributeValueMemberS{Value: id},
		},
		ConditionExpression: aws.String("attribute_exists(id)"),
	})
	if err != nil {
		var condErr *types.ConditionalCheckFailedException
		if errors.As(err, &condErr) {
			return respond(404, map[string]string{"error": "event not found"})
		}
		return respond(500, map[string]string{"error": err.Error()})
	}

	return respond(204, nil)
}
