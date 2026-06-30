output "api_url" {
    description = "API Gateway endpoint"
    value       = "${aws_apigatewayv2_stage.prod.invoke_url}"
}

output "frontend_url" {
    description = "S3 static website URL (origin, not for direct use)"
    value       = "http://${aws_s3_bucket_website_configuration.frontend.website_endpoint}"
}

output "site_url" {
    description = "Public site URL"
    value       = "https://haircuts.1136mpco.com"
}

output "table_name" {
    description = "DynamoDB table name"
    value       = "${aws_dynamodb_table.events.name}"
}