data "archive_file" "cognito_trigger_zip" {
  type        = "zip"
  source_file = "${path.module}/../cognito-trigger/handler.py"
  output_path = "${path.module}/../.terraform/cognito-trigger.zip"
}

resource "aws_iam_role" "cognito_trigger" {
  name = "${var.app_name}-cognito-trigger-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action    = "sts:AssumeRole"
      Effect    = "Allow"
      Principal = { Service = "lambda.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy_attachment" "cognito_trigger_logs" {
  role       = aws_iam_role.cognito_trigger.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "cognito_trigger_policy" {
  name = "${var.app_name}-cognito-trigger-policy"
  role = aws_iam_role.cognito_trigger.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "cognito-idp:AdminAddUserToGroup"
      Resource = aws_cognito_user_pool.main.arn
    }]
  })
}

resource "aws_lambda_function" "cognito_trigger" {
  function_name    = "${var.app_name}-cognito-trigger"
  role             = aws_iam_role.cognito_trigger.arn
  runtime          = "python3.12"
  handler          = "handler.handler"
  filename         = data.archive_file.cognito_trigger_zip.output_path
  source_code_hash = data.archive_file.cognito_trigger_zip.output_base64sha256
  timeout          = 10
}

# Allow Cognito to invoke this Lambda
resource "aws_lambda_permission" "cognito_trigger" {
  statement_id  = "AllowCognitoInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.cognito_trigger.function_name
  principal     = "cognito-idp.amazonaws.com"
  source_arn    = aws_cognito_user_pool.main.arn
}
