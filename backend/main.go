package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

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

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method
	path := req.RawPath

	// /appointments/{id}/cancel — extract id from path manually
	// (path params aren't populated on the $default catch-all route)
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	case method == "GET" && path == "/slots":
		return listSlots(ctx, req)

	case method == "GET" && path == "/services":
		return listServices()

	case method == "POST" && path == "/appointments":
		return bookAppointment(ctx, req)

	case method == "GET" && path == "/appointments/me":
		return myAppointments(ctx, req)

	case method == "PUT" && len(parts) == 3 && parts[0] == "appointments" && parts[2] == "cancel":
		return cancelAppointment(ctx, req, parts[1])

	default:
		return respond(404, map[string]string{"error": "not found"})
	}
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

func main() {
	lambda.Start(handler)
}
