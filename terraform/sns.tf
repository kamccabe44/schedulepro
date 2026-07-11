# ──────────────────────────────────────────────────────────────────────────────
# SNS mobile push platform applications.
#
# Each is created only when its credentials are supplied (count = 0/1), so a
# default deploy with no push credentials is unchanged — no SNS resources exist
# and the Lambda's platform-app ARNs are empty, making push a no-op.
# ──────────────────────────────────────────────────────────────────────────────

locals {
  apns_enabled = var.apns_key != "" && var.apns_key_id != "" && var.apns_team_id != ""
  fcm_enabled  = var.fcm_service_account_json != ""
}

# iOS — Apple Push Notification service (token-based auth with a .p8 key).
resource "aws_sns_platform_application" "apns" {
  count = local.apns_enabled ? 1 : 0

  name                = "${var.app_name}-apns"
  platform            = var.apns_sandbox ? "APNS_SANDBOX" : "APNS"
  platform_credential = var.apns_key    # .p8 signing key contents
  platform_principal  = var.apns_key_id # key ID

  apple_platform_team_id   = var.apns_team_id
  apple_platform_bundle_id = var.apns_bundle_id
}

# Android — Firebase Cloud Messaging (HTTP v1, service-account auth).
resource "aws_sns_platform_application" "fcm" {
  count = local.fcm_enabled ? 1 : 0

  name                = "${var.app_name}-fcm"
  platform            = "GCM"
  platform_credential = var.fcm_service_account_json
}
