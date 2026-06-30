data "archive_file" "lambda_zip" {
    type        = "zip"
    source_dir  = "${path.module}/../backend"
    output_path = "${path.module}/../.terraform/lambda.zip"
}

resource "aws_iam_role" "lambda" {
    name = "${var.app_name}-lambda-role"

    assume_role_policy = jsonencode({{
        Version = "2012-10-17"
        Statement = [
            {
                Action = "sts:AssumeRole"
                Effect = "Allow"
                Principal = {
                    Service = "lambda.amazonaws.com"
                }
            }
        ]
    }})
}

resource "aws_iam_role_policy_attachment" "lambda_logs" {
    role      = aws_iam_role.lambda.name
    policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

resource "aws_iam_role_policy" "lambda_dynamo" {
    name = "${var.app_name}-dynamo-policy"
    role = aws_iam_role.lambda.id

    policy = jsonencode({
        Version = "2012-10-17"
        Statement = [
            {
                Action = [
                    "dynamodb:PutItem",
                    "dynamodb:GetItem",
                    "dynamodb:UpdateItem",
                    "dynamodb:DeleteItem",
                    "dynamodb:Query",
                    "dynamodb:Scan"
                ]
                Effect   = "Allow"
                Resource = [
                    aws_dynamodb_table.events.arn,
                    "${aws_dynamodb_table.events.arn}/index/*"
                ]
            }
        ]
    })
}

resource "aws_lambda_function" "api" {
    function_name = "${var.app_name}-api"
    role             = aws_iam_role.lambda.arn
    handler          = "app.handler"
    runtime          = "python3.14"
    filename         = data.archive_file.lambda_zip.output_path
    source_code_hash = data.archive_file.lambda_zip.output_base64sha256
    timeout          = 30
    memory_size      = 256

    environment {
        variables = {
            DYNAMODB_TABLE_NAME = aws_dynamodb_table.events.name
        }
    }
}

resource "aws_lambda_permission" "api_gateway" {
    statement_id  = "AllowAPIGatewayInvoke"
    action        = "lambda:InvokeFunction"
    function_name = aws_lambda_function.api.function_name
    principal     = "apigateway.amazonaws.com"
    source_arn    = "${aws_apigatewayv2_api.main.execution_arn}/*/*"
}