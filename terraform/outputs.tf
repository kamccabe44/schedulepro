output "api_url" {
  description = "API Gateway endpoint"
  value       = aws_apigatewayv2_stage.prod.invoke_url
}

output "frontend_url" {
  description = "S3 static website URL (origin)"
  value       = "http://${aws_s3_bucket_website_configuration.frontend.website_endpoint}"
}

output "site_url" {
  description = "Public site URL"
  value       = "https://haircuts.1136mpco.com"
}

output "cognito_domain" {
  description = "Cognito Hosted UI base URL"
  value       = "https://${aws_cognito_user_pool_domain.main.domain}.auth.${var.region}.amazoncognito.com"
}

output "cognito_client_id" {
  description = "Cognito App Client ID"
  value       = aws_cognito_user_pool_client.app.id
}

output "cognito_user_pool_id" {
  description = "Cognito User Pool ID"
  value       = aws_cognito_user_pool.main.id
}

output "table_name" {
  description = "DynamoDB appointments table"
  value       = aws_dynamodb_table.appointments.name
}
