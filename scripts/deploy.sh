#!/usr/bin/env bash
set -euo pipefail

STACK_NAME="${STACK_NAME:-schedulepro}"
REGION="${AWS_DEFAULT_REGION:-us-east-1}"

echo "==> Building and deploying SAM stack: $STACK_NAME"
sam build
sam deploy \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --capabilities CAPABILITY_IAM \
  --resolve-s3 \
  --no-confirm-changeset

echo "==> Fetching stack outputs..."
API_URL=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='ApiUrl'].OutputValue" \
  --output text)

BUCKET=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='FrontendUrl'].OutputValue" \
  --output text | sed 's|http://||;s|.s3-website.*||')

FRONTEND_URL=$(aws cloudformation describe-stacks \
  --stack-name "$STACK_NAME" \
  --region "$REGION" \
  --query "Stacks[0].Outputs[?OutputKey=='FrontendUrl'].OutputValue" \
  --output text)

echo "==> Injecting API URL into frontend..."
# Write a config shim so the frontend knows the API endpoint without a build step
cat > /tmp/config.js <<EOF
window.SCHEDULEPRO_API_URL = "$API_URL";
EOF

echo "==> Uploading frontend to S3..."
aws s3 sync frontend/ "s3://$BUCKET/" \
  --region "$REGION" \
  --delete \
  --cache-control "max-age=300"

aws s3 cp /tmp/config.js "s3://$BUCKET/config.js" \
  --region "$REGION" \
  --cache-control "no-cache"

# Inject config.js into index.html if not already present
if ! aws s3 cp "s3://$BUCKET/index.html" - | grep -q "config.js"; then
  aws s3 cp "s3://$BUCKET/index.html" /tmp/index.html
  sed -i 's|<script src="app.js">|<script src="config.js"></script>\n  <script src="app.js">|' /tmp/index.html
  aws s3 cp /tmp/index.html "s3://$BUCKET/index.html" \
    --cache-control "no-cache"
fi

echo ""
echo "✅ Deployment complete!"
echo "   Frontend: $FRONTEND_URL"
echo "   API:      $API_URL"
