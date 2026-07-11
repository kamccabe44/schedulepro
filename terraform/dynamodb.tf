resource "aws_dynamodb_table" "appointments" {
  name         = "${var.app_name}-appointments"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "userId"
    type = "S"
  }

  attribute {
    name = "date"
    type = "S"
  }

  # Query a user's appointments sorted by date
  global_secondary_index {
    name            = "userId-date-index"
    hash_key        = "userId"
    range_key       = "date"
    projection_type = "ALL"
  }

  # Query all appointments for a given date (barber view + slot availability)
  global_secondary_index {
    name            = "date-index"
    hash_key        = "date"
    projection_type = "ALL"
  }
}

resource "aws_dynamodb_table" "barber_settings" {
  name         = "${var.app_name}-barber-settings"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "barberId"

  attribute {
    name = "barberId"
    type = "S"
  }
}

# Push notification device tokens — one user may have several devices, so the
# table is keyed (userId, token). Queried by userId to fan out notifications.
resource "aws_dynamodb_table" "device_tokens" {
  name         = "${var.app_name}-device-tokens"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "userId"
  range_key    = "token"

  attribute {
    name = "userId"
    type = "S"
  }

  attribute {
    name = "token"
    type = "S"
  }
}
