# Alexa Attendance Skill

A voice-controlled attendance tracking system for Amazon Alexa. Record arrivals and departures by simply speaking a person's name, and have everything automatically logged to a Google Sheets spreadsheet.

> "Alexa, open attendance control"
> "Joao Silva arrived"
> "Got it. Joao Silva has been recorded as present."

## How It Works

The skill runs as an AWS Lambda function that receives voice commands from Alexa and writes entries to a Google Sheets spreadsheet. Each day gets its own sheet tab (named by date), with columns for **Name**, **Arrival**, and **Departure**.

A second Lambda function runs daily on a schedule to create the new sheet tab before the workday begins.

### Voice Commands

| Action | Example phrases |
|---|---|
| Record arrival | "Joao arrived", "register Joao's arrival", "mark Joao as present" |
| Record departure | "Joao left", "register Joao's departure", "Joao is leaving" |

When a departure is recorded, the skill finds the person's most recent open arrival row and fills in the departure time. If no arrival exists, it asks for confirmation before recording a departure-only entry.

## Architecture

```
Alexa Device  -->  Alexa Service  -->  Lambda (Go, ARM64)  -->  Google Sheets API
                                            |
                                       AWS Secrets Manager
                                       (Google credentials)

EventBridge (daily cron)  -->  Sheet-Setup Lambda  -->  Google Sheets API
```

**Tech stack:** Go, AWS Lambda, AWS CDK (TypeScript), Google Sheets API v4, Alexa Skills Kit

## Prerequisites

- **AWS account** with permissions to create Lambda functions, IAM roles, and Secrets Manager secrets
- **Amazon Developer account** to create and configure the Alexa skill
- **Google Cloud project** with the Sheets API enabled
- **Google service account** with Editor access to the target spreadsheet
- **Go 1.25+** for building the Lambda binaries
- **Node.js 18+** for CDK deployment (if using CDK)
- **GNU Make** or **PowerShell** (Windows) for running build/deploy commands

## Setup

### 1. Google Sheets

1. Create a new Google Sheets spreadsheet (or use an existing one).
2. Create a Google Cloud service account with the Sheets API enabled.
3. Share the spreadsheet with the service account email (Editor role).
4. Download the service account JSON credentials file.

### 2. AWS Secrets Manager

Store the Google credentials JSON in AWS Secrets Manager:

```bash
make store-secret
# Expects a credentials.json file in the project root
```

### 3. Alexa Skill

1. Create a new custom Alexa skill in the [Alexa Developer Console](https://developer.amazon.com/alexa/console/ask).
2. Set the invocation name to **"controle de presenca"** (or your preferred name).
3. Import the interaction model from `skill-package/interactionModels/pt-BR.json`.
4. Set the skill endpoint to the Lambda function ARN (obtained after deployment).

### 4. Deploy

Three deployment options are available:

#### Option A: AWS CDK (recommended)

```bash
# Build Lambda binaries (run in WSL or Linux for cross-compilation)
make build-all-artifacts

# Deploy the stack
make deploy-cdk \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx
```

#### Option B: CloudFormation

```bash
make deploy-cfn \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx \
  S3_BUCKET=your-deployment-bucket
```

#### Option C: Direct Lambda CLI

```bash
# First-time creation
make create-lambda \
  GOOGLE_SHEET_ID=your-spreadsheet-id \
  ALEXA_SKILL_ID=amzn1.ask.skill.xxx

# Subsequent updates
make deploy-lambda
```

#### PowerShell (Windows)

```powershell
.\deploy.ps1 deploy-cdk -GoogleSheetId "your-spreadsheet-id" -AlexaSkillId "amzn1.ask.skill.xxx"
```

### Configuration

| Environment Variable | Description | Default |
|---|---|---|
| `GOOGLE_SHEET_ID` | Google Sheets spreadsheet ID | *(required)* |
| `ALEXA_SKILL_ID` | Alexa Skill ID for request validation | *(optional)* |
| `GOOGLE_CREDENTIALS_SECRET` | Secrets Manager secret name | `alexa-skill/google-credentials` |
| `TZ` | Timezone for timestamps | `America/Sao_Paulo` |

| Make Variable | Description | Default |
|---|---|---|
| `REGION` | AWS region | `eu-west-1` |
| `FUNCTION_NAME` | Lambda function name | `alexa-attendance-skill` |

## Project Structure

```
alexa-attendance-skill/
├── cmd/
│   ├── lambda/          # Alexa skill Lambda entry point
│   └── sheet-setup/     # Daily sheet creation Lambda
├── internal/
│   ├── alexa/           # Alexa request/response handling
│   ├── sheets/          # Google Sheets API client
│   └── awsutil/         # AWS Secrets Manager integration
├── infra/
│   ├── cdk/             # AWS CDK stack (TypeScript)
│   └── cloudformation/  # CloudFormation template
├── skill-package/       # Alexa skill manifest and interaction model
├── Makefile             # Build and deploy targets
└── deploy.ps1           # PowerShell deployment script
```

## Testing

```bash
go test ./...
```

The test suite covers Alexa request handling (intent routing, session state, validation) and Google Sheets operations (row creation, updates, sheet setup) using mock interfaces.

## License

This project is provided as-is for personal and educational use.
