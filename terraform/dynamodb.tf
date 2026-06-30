resource "aws_dynamodb_table" "events" {
    name           = "${var.app_name}-events"
    billing_mode   = "PAY_PER_REQUEST"
    hash_key       = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "startDate"
    type = "S"
  }

  global_secondary_index {
    name               = "startDate-index"
    hash_key           = "startDate"
    projection_type    = "ALL"
  }