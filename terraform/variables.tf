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

variable "environment" {
  description = "Deployment environment, used for cost allocation tagging"
  type        = string
  default     = "production"
}

variable "customer_name" {
  description = "Name of the customer this deployment belongs to, used for cost allocation tagging"
  type        = string
}

# ── Push notifications (optional) ──────────────────────────────────────────────
# Leave these empty to deploy exactly as before — no SNS platform applications
# are created and the /devices endpoint stores tokens but sends nothing. Fill
# them in once you have Apple (APNs) and/or Google (FCM) credentials to turn on
# push. See mobile/README.md → "Push notifications".

variable "apns_key" {
  description = "APNs auth key (.p8 contents) for iOS push. Empty disables iOS push."
  type        = string
  default     = ""
  sensitive   = true
}

variable "apns_key_id" {
  description = "APNs key ID (from the Apple Developer key)."
  type        = string
  default     = ""
}

variable "apns_team_id" {
  description = "Apple Developer team ID."
  type        = string
  default     = ""
}

variable "apns_bundle_id" {
  description = "iOS app bundle id / topic (matches capacitor.config.json appId)."
  type        = string
  default     = "com.ozarks.schedulepro"
}

variable "apns_sandbox" {
  description = "Use the APNs sandbox environment (true for dev/TestFlight builds)."
  type        = bool
  default     = true
}

variable "fcm_service_account_json" {
  description = "Firebase service account JSON for Android (FCM v1). Empty disables Android push."
  type        = string
  default     = ""
  sensitive   = true
}