terraform {
    required_version = ">= 1.7"

  required_providers {
    aws = {
        source  = "hashicorp/aws"
        version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# ACM certificates for CloudFront must live in us-east-1 regardless of app region
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

data "aws_caller_identity" "current" {}