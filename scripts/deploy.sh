#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Customer name drives cost-allocation tags and on-site branding.
# Accept it as the first arg, or an existing CUSTOMER_NAME env var, else prompt.
CUSTOMER_NAME="${1:-${CUSTOMER_NAME:-}}"
if [ -z "$CUSTOMER_NAME" ]; then
  read -r -p "Customer name (used for tags and site branding): " CUSTOMER_NAME
fi
if [ -z "$CUSTOMER_NAME" ]; then
  echo "Error: customer name is required." >&2
  exit 1
fi

echo "==> Building Go Lambda binary..."
(
  cd backend
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
)

cd terraform

echo "==> Initializing Terraform..."
terraform init

echo "==> Applying infrastructure..."
terraform apply -auto-approve -var "customer_name=$CUSTOMER_NAME"

API_URL=$(terraform output -raw api_url)
COGNITO_DOMAIN=$(terraform output -raw cognito_domain)
COGNITO_CLIENT_ID=$(terraform output -raw cognito_client_id)
BUCKET=$(terraform output -raw frontend_url | sed 's|http://||;s|\.s3-website.*||')
FRONTEND_URL=$(terraform output -raw site_url)

echo "==> Uploading frontend..."

# Write runtime config for the browser app
cat > /tmp/config.js <<EOF
window.SCHEDPRO_CONFIG = {
  apiUrl:             "$API_URL",
  cognitoDomain:      "$COGNITO_DOMAIN",
  cognitoClientId:    "$COGNITO_CLIENT_ID",
  cognitoRedirectUri: "$FRONTEND_URL",
  siteName:           "$CUSTOMER_NAME",
};
EOF

aws s3 sync ../frontend/ "s3://$BUCKET/" --delete --cache-control "max-age=300"
aws s3 cp /tmp/config.js "s3://$BUCKET/config.js" --cache-control "no-cache"

echo ""
echo "✅ Done!"
echo "   Customer: $CUSTOMER_NAME"
echo "   Site: $FRONTEND_URL"
echo "   API:  $API_URL"
