resource "aws_cognito_user_pool" "main" {
  name = "${var.app_name}-users"

  username_attributes      = ["email"]
  auto_verified_attributes = ["email"]

  password_policy {
    minimum_length    = 8
    require_lowercase = true
    require_numbers   = true
    require_symbols   = false
    require_uppercase = false
  }

  schema {
    attribute_data_type = "String"
    name                = "email"
    required            = true
    mutable             = true
    string_attribute_constraints {
      min_length = 1
      max_length = 256
    }
  }

  schema {
    attribute_data_type = "String"
    name                = "name"
    required            = true
    mutable             = true
    string_attribute_constraints {
      min_length = 1
      max_length = 100
    }
  }

  # Auto-assign new signups to the customers group
  lambda_config {
    post_confirmation = aws_lambda_function.cognito_trigger.arn
  }
}

resource "aws_cognito_user_pool_domain" "main" {
  domain       = "${var.app_name}-${data.aws_caller_identity.current.account_id}"
  user_pool_id = aws_cognito_user_pool.main.id
}

resource "aws_cognito_user_pool_client" "app" {
  name         = "${var.app_name}-client"
  user_pool_id = aws_cognito_user_pool.main.id

  generate_secret = false

  allowed_oauth_flows_user_pool_client = true
  allowed_oauth_flows                  = ["code"]
  allowed_oauth_scopes                 = ["openid", "email", "profile"]

  supported_identity_providers = ["COGNITO"]

  callback_urls = [
    "https://haircuts.1136mpco.com",
    "http://localhost:8080",
  ]

  logout_urls = [
    "https://haircuts.1136mpco.com",
    "http://localhost:8080",
  ]

  explicit_auth_flows = [
    "ALLOW_USER_SRP_AUTH",
    "ALLOW_REFRESH_TOKEN_AUTH",
  ]
}

resource "aws_cognito_user_group" "customers" {
  name         = "customers"
  user_pool_id = aws_cognito_user_pool.main.id
  description  = "Customers who can book appointments"
  precedence   = 10
}

resource "aws_cognito_user_group" "barbers" {
  name         = "barbers"
  user_pool_id = aws_cognito_user_pool.main.id
  description  = "Barbers who can view all appointments"
  precedence   = 5
}

resource "aws_cognito_user_group" "admins" {
  name         = "admins"
  user_pool_id = aws_cognito_user_pool.main.id
  description  = "Admins who can manage barbers and settings"
  precedence   = 1
}
