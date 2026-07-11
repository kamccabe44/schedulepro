// ─────────────────────────────────────────────────────────────────────────────
// Mobile build configuration
//
// On the web, this file is generated at deploy time and served from S3. Native
// apps have no S3 to fetch from, so the values are BAKED INTO the app bundle
// here. Copy your PRODUCTION values from `scripts/deploy.sh` output (or the
// Terraform outputs: api_url, cognito_domain, cognito_client_id).
//
// IMPORTANT — Cognito redirect URI:
// A native app cannot redirect back to a web URL. It must use a custom scheme
// that the OS routes back into the app. Set `cognitoRedirectUri` below to the
// custom scheme AND register that exact value as an allowed callback URL in the
// Cognito app client (see mobile/README.md → "Auth on native").
// ─────────────────────────────────────────────────────────────────────────────
window.SCHEDPRO_CONFIG = {
  apiUrl: "https://REPLACE-WITH-YOUR-API-ID.execute-api.us-east-1.amazonaws.com",
  cognitoDomain: "https://REPLACE-WITH-YOUR-DOMAIN.auth.us-east-1.amazoncognito.com",
  cognitoClientId: "REPLACE-WITH-YOUR-COGNITO-CLIENT-ID",

  // Native custom-scheme callback — must match capacitor.config.json appId.
  cognitoRedirectUri: "com.ozarks.schedulepro://oauth-callback",

  siteName: "SchedulePro",

  // Marks this as a native build so app.js / helpers can branch if needed.
  platform: "native",
};
