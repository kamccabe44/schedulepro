package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var (
	db           *dynamodb.Client
	cognitoClient *cognitoidentityprovider.Client
	tableName    string
	stageName    string
	userPoolID   string
)

func init() {
	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		panic("failed to load AWS config: " + err.Error())
	}
	db = dynamodb.NewFromConfig(cfg)
	cognitoClient = cognitoidentityprovider.NewFromConfig(cfg)
	tableName = os.Getenv("TABLE_NAME")
	stageName = os.Getenv("STAGE_NAME")
	userPoolID = os.Getenv("USER_POOL_ID")
}

func handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	method := req.RequestContext.HTTP.Method

	// API Gateway includes the stage name in rawPath for named stages.
	// Strip it so route matching works correctly.
	path := strings.TrimPrefix(req.RawPath, "/"+stageName)
	if path == "" {
		path = "/"
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	case method == "OPTIONS":
		return respond(200, nil)

	// ── Public ────────────────────────────────────────────────────────────
	case method == "GET" && path == "/slots":
		return listSlots(ctx, req)
	case method == "GET" && path == "/services":
		return listServices()

	// ── Customer ──────────────────────────────────────────────────────────
	case method == "POST" && path == "/appointments":
		return bookAppointment(ctx, req)
	case method == "GET" && path == "/appointments/me":
		return myAppointments(ctx, req)
	case method == "PUT" && len(parts) == 3 && parts[0] == "appointments" && parts[2] == "cancel":
		return cancelAppointment(ctx, req, parts[1])

	// ── Barber / Admin ────────────────────────────────────────────────────
	case method == "GET" && path == "/admin/appointments":
		return adminAppointments(ctx, req)

	// ── Admin only ────────────────────────────────────────────────────────
	case method == "GET" && path == "/admin/barbers":
		return listBarbers(ctx, req)
	case method == "POST" && path == "/admin/barbers":
		return addBarber(ctx, req)
	case method == "DELETE" && len(parts) == 3 && parts[0] == "admin" && parts[1] == "barbers":
		return removeBarber(ctx, req, parts[2])

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
