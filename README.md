# SchedulePro

Minimal-cost AWS scheduling application. Month/week/day calendar with full CRUD — no server to maintain, costs near $0 for light use.

## Architecture

```
Browser → S3 (static HTML/JS/CSS)
             ↕ fetch
       API Gateway HTTP API (cheapest API type)
             ↕
          Lambda (Python 3.12)
             ↕
          DynamoDB (on-demand billing)
```

**AWS free tier covers almost everything:**
- Lambda: 1M requests/month free (always)
- DynamoDB: 25 GB storage + 200M requests/month free (always)
- API Gateway HTTP API: $1/million requests after free tier
- S3: ~$0.023/GB/month

## Prerequisites

- [AWS CLI](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html) configured (`aws configure`)
- [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html)
- Python 3.12

## Deploy

```bash
# One-command deploy
./scripts/deploy.sh

# Or manually:
sam build
sam deploy --guided   # first time — walks you through config
```

The deploy script:
1. Builds and deploys the Lambda + DynamoDB + API Gateway via SAM
2. Uploads the frontend to S3
3. Injects the API URL into the frontend automatically

## Tear down

```bash
aws cloudformation delete-stack --stack-name schedulepro
```

## Customization

| What | Where |
|---|---|
| App name / region | `samconfig.toml` |
| Lambda memory/timeout | `template.yaml` → `Globals` |
| DynamoDB table name | `template.yaml` → `Parameters.AppName` |
| Event fields / validation | `backend/app.py` → `create_event`, `validate_event` |
| Calendar UI | `frontend/app.js`, `frontend/styles.css` |
| Color palette | `frontend/index.html` → `.color-options` |

## Local development

```bash
# Start a local DynamoDB (requires Docker)
docker run -p 8000:8000 amazon/dynamodb-local

# Create the local table
aws dynamodb create-table \
  --table-name schedulepro-events \
  --attribute-definitions AttributeName=id,AttributeType=S AttributeName=startDate,AttributeType=S \
  --key-schema AttributeName=id,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --global-secondary-indexes '[{"IndexName":"startDate-index","KeySchema":[{"AttributeName":"startDate","KeyType":"HASH"}],"Projection":{"ProjectionType":"ALL"}}]' \
  --endpoint-url http://localhost:8000

# Run the API locally (hot-reload)
TABLE_NAME=schedulepro-events sam local start-api

# Open frontend/index.html directly in a browser
# Change API_URL in app.js to http://127.0.0.1:3000 for local testing
```

## Project structure

```
schedulepro/
├── template.yaml        # SAM / CloudFormation infrastructure
├── samconfig.toml       # Deploy defaults (region, stack name)
├── backend/
│   ├── app.py           # Lambda handler — all API routes
│   └── requirements.txt
├── frontend/
│   ├── index.html       # Calendar shell
│   ├── styles.css       # Styles (no framework)
│   └── app.js           # Calendar logic + API client
└── scripts/
    └── deploy.sh        # One-shot build + deploy + frontend upload
```
