# Alexa Attendance Skill

Voice-based attendance tracking for Amazon Alexa. Say a person's name, record their arrival or departure, and keep the log in Google Sheets without touching a keyboard.

## Example Conversation

> "Alexa, open attendance control"
> "Joao Silva arrived"
> "Got it. Joao Silva has been recorded as present."

## Overview

The skill runs on AWS Lambda and writes attendance events to Google Sheets. Each day gets its own tab, with columns for **Name**, **Arrival**, and **Departure**.

A second Lambda function runs on a schedule to create the next daily sheet automatically before the workday starts.

### Supported Intents

| Action | Example phrases |
|---|---|
| Record arrival | "Joao arrived", "register Joao's arrival", "mark Joao as present" |
| Record departure | "Joao left", "register Joao's departure", "Joao is leaving" |

When a departure is recorded, the skill updates the most recent open arrival row for that person. If no arrival exists, Alexa asks for confirmation before saving a departure-only entry.

## Architecture

```text
Alexa Device  ->  Alexa Service  ->  Lambda (Go, ARM64)  ->  Google Sheets API
                                           |
                                      AWS Secrets Manager
                                      (Google credentials)

EventBridge (daily cron)  ->  Sheet-Setup Lambda  ->  Google Sheets API
```

**Tech stack:** Go, AWS Lambda, AWS CDK (TypeScript), Google Sheets API v4, Alexa Skills Kit

## Prerequisites

- AWS account with permissions for Lambda, IAM, EventBridge, CloudFormation, and Secrets Manager
- Amazon Developer account for the Alexa skill
- Google Cloud project with the Sheets API enabled
- Google service account with Editor access to the target spreadsheet
- Go 1.25+
- Node.js 18+
- GNU Make or PowerShell
- ASK CLI for syncing the interaction model from the command line

## Recommended Setup Flow

### 1. Prepare Google Sheets

1. Create the target spreadsheet.
2. Create a Google Cloud service account with the Sheets API enabled.
3. Share the spreadsheet with the service account email as **Editor**.
4. Download the service account JSON file as `credentials.json`.

### 2. Store `credentials.json` in AWS Secrets Manager

Create the secret the first time:

```bash
aws secretsmanager create-secret \
  --name alexa-skill/google-credentials \
  --secret-string file://credentials.json \
  --region sa-east-1
```

Update the secret later if it already exists:

```bash
aws secretsmanager put-secret-value \
  --secret-id alexa-skill/google-credentials \
  --secret-string file://credentials.json \
  --region sa-east-1
```

If you prefer the helper target, `make store-secret` wraps the first-time creation flow.

### 3. Create and Configure the Alexa Skill

1. Create a custom Alexa skill in the [Alexa Developer Console](https://developer.amazon.com/alexa/console/ask).
2. Set the invocation name to `controle de presenca` or your preferred variation.
3. Use the interaction model from `skill-package/interactionModels/pt-BR.json`.
4. After deployment, point the skill endpoint to the Lambda ARN.

### 4. Build the Lambda Artifacts

```bash
make build-all-artifacts
```

Run this from Linux or WSL so the Lambda binaries are built for `linux/arm64`.

### 5. Deploy with CDK

The CDK workflow is the recommended path.

If this is a completely new AWS account/region deployment, bootstrap CDK first:

```bash
cd infra/cdk
npx cdk bootstrap aws://ACCOUNT_ID/sa-east-1
```

Deploy the stack:

```bash
make deploy-cdk \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx
```

> Important: whenever you run raw CDK app commands such as `cdk synth`, `cdk diff`, or `cdk deploy`, do not forget to pass `-c googleSheetId=xxx`. The CDK app expects that context value.

Example:

```bash
cd infra/cdk
npx cdk diff \
  -c googleSheetId=your-spreadsheet-id \
  -c alexaSkillId=amzn1.ask.skill.xxx \
  -c googleCredentialsSecretName=alexa-skill/google-credentials \
  -c region=sa-east-1
```

### 6. Sync the Interaction Model with ASK CLI

After creating the skill, or whenever `skill-package/interactionModels/pt-BR.json` changes, push the latest model with ASK CLI:

```bash
ask smapi set-interaction-model -s amzn1.ask.skill.id -g development -l pt-BR --interaction-model "$(cat skill-package/interactionModels/pt-BR.json)"
```

Replace `amzn1.ask.skill.id` with your real skill ID.

## Other Deployment Options

### CloudFormation

```bash
make deploy-cfn \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx \
  S3_BUCKET=your-deployment-bucket
```

### Direct Lambda CLI

```bash
# First-time creation
make create-lambda \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx

# Subsequent updates
make deploy-lambda
```

### PowerShell

```powershell
.\deploy.ps1 deploy-cdk -GoogleSheetId "your-spreadsheet-id" -AlexaSkillId "amzn1.ask.skill.xxx"
```

## Configuration

### Runtime Environment Variables

| Variable | Description | Default |
|---|---|---|
| `GOOGLE_SHEET_ID` | Google Sheets spreadsheet ID | required |
| `ALEXA_SKILL_ID` | Alexa skill ID used for request validation | optional |
| `GOOGLE_CREDENTIALS_SECRET` | Secrets Manager secret name | `alexa-skill/google-credentials` |
| `TZ` | Time zone used for timestamps | `America/Sao_Paulo` |

### Common Make Variables

| Variable | Description | Default |
|---|---|---|
| `REGION` | AWS region | `sa-east-1` |
| `FUNCTION_NAME` | Main Lambda function name | `alexa-attendance-skill` |
| `SECRET_NAME` | Secrets Manager secret name | `alexa-skill/google-credentials` |

## Project Structure

```text
alexa-attendance-skill/
|-- cmd/
|   |-- lambda/          # Alexa skill Lambda entry point
|   `-- sheet-setup/     # Daily sheet creation Lambda
|-- internal/
|   |-- alexa/           # Alexa request and response handling
|   |-- awsutil/         # AWS Secrets Manager integration
|   `-- sheets/          # Google Sheets client logic
|-- infra/
|   |-- cdk/             # AWS CDK stack
|   `-- cloudformation/  # CloudFormation template
|-- skill-package/       # Alexa skill manifest and interaction model
|-- Makefile             # Build and deploy commands
`-- deploy.ps1           # PowerShell deployment helper
```

## Testing

```bash
go test ./...
```

The test suite covers Alexa request handling, session flows, validation, and Google Sheets operations through mocks.

## License

This project is provided as-is for personal and educational use.
