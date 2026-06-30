variable "app_name" {
    description = "Prefix for all resource names"
    type        = string
    default     = "schedpro"
}

variable "region" {
    description = "AWS region to deploy resources"
    type        = string
    default     = "us-east-1"
}