resource "aws_apigatewayv2_api" "http_api" {
    name          = "${var.app_name}-api"
    protocol_type = "HTTP"

    cors_configuration {
        allow_origins = ["*"]
        allow_headers = ["Content-Type", "Authorization"]
        allow_methods = ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    }
}

resource "aws_apigatewayv2_integration" "lambda" {
    api_id           = aws_apigatewayv2_api.http_api.id
    integration_type = "AWS_PROXY"
    integration_uri  = aws_lambda_function.api.invoke_arn
    payload_format_version = "2.0"
}

resource "aws_apigatewayv2_route" "proxy" {
    api_id    = aws_apigatewayv2_api.http_api.id
    route_key = "$default"
    target    = "integrations/${aws_apigatewayv2_integration.lambda.id}"
}

resource "aws_apigatewayv2_stage" "prod" {
    api_id      = aws_apigatewayv2_api.http_api.id
    name        = "prod"
    auto_deploy = true
}