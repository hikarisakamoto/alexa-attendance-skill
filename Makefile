# ──────────────────────────────────────────────────
# Lambda
# ──────────────────────────────────────────────────

FUNCTION_NAME ?= alexa-attendance-skill
REGION        ?= sa-east-1
ROLE_NAME     ?= alexa-skill-lambda-role
SECRET_NAME   ?= alexa-skill/google-credentials

# Build the Lambda binary and zip it
build-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
	@if command -v zip >/dev/null 2>&1; then \
		chmod +x bootstrap && zip function.zip bootstrap; \
	elif command -v python3 >/dev/null 2>&1; then \
		python3 -c "import zipfile; z=zipfile.ZipFile('function.zip','w',zipfile.ZIP_DEFLATED); i=zipfile.ZipInfo('bootstrap'); i.external_attr=0o755<<16; z.writestr(i,open('bootstrap','rb').read()); z.close()"; \
	elif command -v python >/dev/null 2>&1; then \
		python -c "import zipfile; z=zipfile.ZipFile('function.zip','w',zipfile.ZIP_DEFLATED); i=zipfile.ZipInfo('bootstrap'); i.external_attr=0o755<<16; z.writestr(i,open('bootstrap','rb').read()); z.close()"; \
	else \
		echo "Error: no zip tool found. Install zip or python3." >&2; exit 1; \
	fi
	rm bootstrap

# First-time setup: create IAM role, Lambda function, and Alexa trigger.
# Usage: make create-lambda GOOGLE_SHEET_ID=your-sheet-id ALEXA_SKILL_ID=amzn1.ask.skill.xxx
create-lambda: build-lambda
	@echo "Creating IAM role..."
	aws iam create-role \
		--role-name $(ROLE_NAME) \
		--assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}' \
		--region $(REGION)
	aws iam attach-role-policy \
		--role-name $(ROLE_NAME) \
		--policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
	@echo "Waiting for role to propagate..."
	sleep 10
	@echo "Creating Lambda function..."
	aws lambda create-function \
		--function-name $(FUNCTION_NAME) \
		--runtime provided.al2023 \
		--architectures arm64 \
		--handler bootstrap \
		--role $$(aws iam get-role --role-name $(ROLE_NAME) --query 'Role.Arn' --output text) \
		--zip-file fileb://function.zip \
		--environment "Variables={GOOGLE_SHEET_ID=$(GOOGLE_SHEET_ID),GOOGLE_CREDENTIALS_SECRET=$(SECRET_NAME),ALEXA_SKILL_ID=$(ALEXA_SKILL_ID),TZ=America/Sao_Paulo}" \
		--timeout 15 \
		--memory-size 128 \
		--region $(REGION)
	@echo "Adding Alexa Skills Kit trigger..."
	aws lambda add-permission \
		--function-name $(FUNCTION_NAME) \
		--statement-id alexa-skill-trigger \
		--action lambda:InvokeFunction \
		--principal alexa-appkit.amazon.com \
		--event-source-token $(ALEXA_SKILL_ID) \
		--region $(REGION)
	@echo "Adding Secrets Manager permission..."
	aws iam put-role-policy \
		--role-name $(ROLE_NAME) \
		--policy-name SecretsManagerRead \
		--policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"secretsmanager:GetSecretValue","Resource":"arn:aws:secretsmanager:$(REGION):*:secret:$(SECRET_NAME)-*"}]}'
	rm function.zip
	@echo "Done! Lambda ARN:"
	@aws lambda get-function --function-name $(FUNCTION_NAME) --query 'Configuration.FunctionArn' --output text --region $(REGION)

# Deploy updated code to an existing Lambda function
deploy-lambda: build-lambda
	aws lambda update-function-code \
		--function-name $(FUNCTION_NAME) \
		--zip-file fileb://function.zip \
		--region $(REGION)
	rm function.zip

# Store Google credentials in Secrets Manager
store-secret:
	aws secretsmanager create-secret \
		--name "$(SECRET_NAME)" \
		--description "Google Sheets service account credentials for Alexa attendance skill" \
		--secret-string file://credentials.json \
		--region $(REGION)

# ──────────────────────────────────────────────────
# CloudFormation
# ──────────────────────────────────────────────────

S3_KEY ?= alexa-skill/function.zip
SHEET_SETUP_S3_KEY ?= alexa-skill/sheet-setup.zip

# Build the sheet-setup Lambda binary and zip it
build-sheet-setup-lambda:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/sheet-setup
	@if command -v zip >/dev/null 2>&1; then \
		chmod +x bootstrap && zip sheet-setup.zip bootstrap; \
	elif command -v python3 >/dev/null 2>&1; then \
		python3 -c "import zipfile; z=zipfile.ZipFile('sheet-setup.zip','w',zipfile.ZIP_DEFLATED); i=zipfile.ZipInfo('bootstrap'); i.external_attr=0o755<<16; z.writestr(i,open('bootstrap','rb').read()); z.close()"; \
	elif command -v python >/dev/null 2>&1; then \
		python -c "import zipfile; z=zipfile.ZipFile('sheet-setup.zip','w',zipfile.ZIP_DEFLATED); i=zipfile.ZipInfo('bootstrap'); i.external_attr=0o755<<16; z.writestr(i,open('bootstrap','rb').read()); z.close()"; \
	else \
		echo "Error: no zip tool found. Install zip or python3." >&2; exit 1; \
	fi
	rm bootstrap

# Usage: make deploy-cfn GOOGLE_SHEET_ID=xxx ALEXA_SKILL_ID=amzn1.ask.skill.xxx S3_BUCKET=your-bucket
deploy-cfn: build-lambda build-sheet-setup-lambda
	aws s3 cp function.zip s3://$(S3_BUCKET)/$(S3_KEY) --region $(REGION)
	aws s3 cp sheet-setup.zip s3://$(S3_BUCKET)/$(SHEET_SETUP_S3_KEY) --region $(REGION)
	aws cloudformation deploy \
		--template-file infra/cloudformation/template.yaml \
		--stack-name alexa-attendance-skill \
		--capabilities CAPABILITY_NAMED_IAM \
		--region $(REGION) \
		--parameter-overrides \
			GoogleSheetId=$(GOOGLE_SHEET_ID) \
			AlexaSkillId=$(ALEXA_SKILL_ID) \
			GoogleCredentialsSecretName=$(SECRET_NAME) \
			LambdaCodeS3Bucket=$(S3_BUCKET) \
			LambdaCodeS3Key=$(S3_KEY) \
			SheetSetupCodeS3Key=$(SHEET_SETUP_S3_KEY)
	rm function.zip sheet-setup.zip

# ──────────────────────────────────────────────────
# CDK
# ──────────────────────────────────────────────────

ARTIFACT_DIR ?= dist/lambda

# Build the Lambda binary into a directory (for CDK artifactPath).
# Run this in WSL, then use the artifact dir with CDK deploy from PowerShell.
build-artifact:
	mkdir -p $(ARTIFACT_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o $(ARTIFACT_DIR)/bootstrap ./cmd/lambda
	chmod +x $(ARTIFACT_DIR)/bootstrap

# Build the sheet-setup Lambda binary into a directory (for CDK sheetSetupArtifactPath).
SHEET_SETUP_ARTIFACT_DIR ?= dist/sheet-setup

build-sheet-setup-artifact:
	mkdir -p $(SHEET_SETUP_ARTIFACT_DIR)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -o $(SHEET_SETUP_ARTIFACT_DIR)/bootstrap ./cmd/sheet-setup
	chmod +x $(SHEET_SETUP_ARTIFACT_DIR)/bootstrap

# Build both Lambda binaries (Alexa skill + sheet setup).
build-all-artifacts: build-artifact build-sheet-setup-artifact

# Usage: make deploy-cdk GOOGLE_SHEET_ID=xxx ALEXA_SKILL_ID=amzn1.ask.skill.xxx [ARTIFACT_PATH=path/to/dir] [SHEET_SETUP_ARTIFACT_PATH=path/to/dir]
deploy-cdk:
	cd infra/cdk && npm install && npx cdk deploy \
		--context googleSheetId=$(GOOGLE_SHEET_ID) \
		--context alexaSkillId=$(ALEXA_SKILL_ID) \
		--context googleCredentialsSecretName=$(SECRET_NAME) \
		--context region=$(REGION) \
		$(if $(ARTIFACT_PATH),--context artifactPath=$(ARTIFACT_PATH)) \
		$(if $(SHEET_SETUP_ARTIFACT_PATH),--context sheetSetupArtifactPath=$(SHEET_SETUP_ARTIFACT_PATH)) \
		--require-approval never

# ──────────────────────────────────────────────────
# Docker (existing workflow)
# ──────────────────────────────────────────────────

build-server:
	docker compose build

run-server:
	docker compose up

.PHONY: build-lambda build-sheet-setup-lambda create-lambda deploy-lambda store-secret build-artifact build-sheet-setup-artifact build-all-artifacts deploy-cfn deploy-cdk build-server run-server
