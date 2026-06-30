output "api_url" {
    description = "API Gateway endpoint"
    value       = "${aws_apigatewayv2_stage.prod.invoke_url}"
}

output "frontend_url" {
    description = "S3 static website URL"
    value       = "http://${aws_s3_bucket.frontend.website_endpoint}"
}

output "table_name" {
    description = "DynamoDB table name"
    value       = "${aws_dynamodb_table.events.name}"
}