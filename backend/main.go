package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var db *dynamodb.Client
var tableName string

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}
	db = dynamodb.NewFromConfig(cfg)
	tableName = os.Getenv("TABLE_NAME")
}

func respond(status int, body any) (events.APIGatewayV2HTTPResponse, error) {
	b, _ := json.Marshal(body)
	return events.APIGatewayV2HTTPResponse{
		StatusCode: status,
		Headers: map[string]string{
			"Content-Type":                "application/json",
			"Access-Control-Allow-Origin": "*",
		},
		Body: string(b),
	}, nil
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RequestContext.HTTP.Path
	id := req.PathParameters["id"]

	switch {
	case method == "GET" && path == "/events":
		return listEvents(ctx, req)
	case method == "POST" && path == "/events":
		return createEvent(ctx, req)
	case method == "GET" && id != "":
		return getEvent(ctx, id)
	case method == "PUT" && id != "":
		return updateEvent(ctx, req, id)
	case method == "DELETE" && id != "":
		return deleteEvent(ctx, id)
	default:
		return respond(404, map[string]string{"error": "not found"})
	}
}

func main() {
	lambda.Start(handler)
}
