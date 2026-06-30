#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/../terraform"

echo "==> Initializing Terraform..."
terraform init 

echo "==> Applying infrastructure..."
terraform apply -auto-approve

API_URL=$(terraform output -raw api_url)
BUCKET=$(terraform output -raw frontend_url | sed 's|http://||;s|\.s3-website.*||')
FRONTEND_URL=$(terraform output -raw frontend_url)

echo "==> Uploading frontend..."
echo "window.SCHEDULEPRO_API_URL = '\"$API_URL\"; > /tmp/config.js"

aws s3 sync ../frontend/ "s3://$BUCKET/" --delete --cache-control "max-age=300"
aws s3 cp /tmp/config.js "s3://$BUCKET/config.js" --cache-control "no-cache"

aws s3 cp "s3://$BUCKET/index.html" /tmp/index.html
if ! grep -q "config.js" /tmp/index.html; then 
  sed -i 's|<script src="app.js>|<script src="config.js"></script>\n <script src="app.js">|' /tmp/index.html
  aws s3 cp /tmp/index.html "s3://$BUCKET/index.html" --cache-control "no-cache"
fi

echo ""
echo "Done!"
echo "  Frontend: $FRONTEND_URL"
echo "  API: $API_URL"