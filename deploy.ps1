# ──────────────────────────────────────────────────
# Alexa Attendance Skill — PowerShell Deploy Script
# ──────────────────────────────────────────────────
#
# Usage:
#   .\deploy.ps1 build-lambda
#   .\deploy.ps1 create-lambda -GoogleSheetId "your-id" -AlexaSkillId "amzn1.ask.skill.xxx"
#   .\deploy.ps1 deploy-lambda
#   .\deploy.ps1 store-secret
#   .\deploy.ps1 build-server
#   .\deploy.ps1 run-server
#   .\deploy.ps1 deploy-cfn -GoogleSheetId "your-id" -AlexaSkillId "amzn1.ask.skill.xxx" -S3Bucket "your-bucket"
#   .\deploy.ps1 deploy-cdk -GoogleSheetId "your-id" -AlexaSkillId "amzn1.ask.skill.xxx"
#   .\deploy.ps1 deploy-cdk -GoogleSheetId "your-id" -AlexaSkillId "amzn1.ask.skill.xxx" -ArtifactPath "../../dist/lambda"
#   .\deploy.ps1 build-artifact
#   .\deploy.ps1 build-sheet-setup-artifact
#   .\deploy.ps1 build-all-artifacts
#   .\deploy.ps1 deploy-cdk -GoogleSheetId "your-id" -SheetSetupArtifactPath "../../dist/sheet-setup"

param(
    [Parameter(Position = 0, Mandatory = $true)]
    [ValidateSet(
        "build-lambda", "build-artifact", "build-sheet-setup-artifact", "build-all-artifacts",
        "create-lambda", "deploy-lambda", "store-secret",
        "build-server", "run-server",
        "deploy-cfn", "deploy-cdk"
    )]
    [string]$Command,

    [string]$FunctionName = "alexa-attendance-skill",
    [string]$Region = "eu-west-1",
    [string]$RoleName = "alexa-skill-lambda-role",
    [string]$SecretName = "alexa-skill/google-credentials",
    [string]$GoogleSheetId,
    [string]$AlexaSkillId,
    [string]$S3Bucket,
    [string]$S3Key = "alexa-skill/function.zip",
    [string]$ArtifactPath,
    [string]$SheetSetupArtifactPath
)

$ErrorActionPreference = "Stop"

function Build-Lambda {
    Write-Host "Building Lambda binary (linux/arm64)..." -ForegroundColor Cyan

    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    try {
        go build -tags lambda.norpc -o bootstrap ./cmd/lambda
        if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
    }
    finally {
        Remove-Item Env:\GOOS
        Remove-Item Env:\GOARCH
        Remove-Item Env:\CGO_ENABLED
    }

    # Use Python to create a zip with Unix execute permissions (0755).
    # PowerShell's Compress-Archive does not set Unix permissions, which
    # causes Runtime.InvalidEntrypoint on Lambda.
    python -c @"
import zipfile
z = zipfile.ZipFile('function.zip', 'w', zipfile.ZIP_DEFLATED)
i = zipfile.ZipInfo('bootstrap')
i.external_attr = 0o755 << 16
z.writestr(i, open('bootstrap', 'rb').read())
z.close()
"@
    if ($LASTEXITCODE -ne 0) { throw "Failed to create function.zip with correct permissions" }
    Remove-Item bootstrap

    Write-Host "Build complete: function.zip" -ForegroundColor Green
}

function Build-Artifact {
    Write-Host "Building Lambda binary into dist/lambda/ (linux/arm64)..." -ForegroundColor Cyan

    if (-not (Test-Path "dist/lambda")) {
        New-Item -ItemType Directory -Path "dist/lambda" -Force | Out-Null
    }

    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    try {
        go build -tags lambda.norpc -o dist/lambda/bootstrap ./cmd/lambda
        if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
    }
    finally {
        Remove-Item Env:\GOOS
        Remove-Item Env:\GOARCH
        Remove-Item Env:\CGO_ENABLED
    }

    Write-Host "Build complete: dist/lambda/bootstrap" -ForegroundColor Green
    Write-Host "Note: For CDK deploy, use: .\deploy.ps1 deploy-cdk -ArtifactPath '../../dist/lambda' ..." -ForegroundColor Yellow
    Write-Host "      On Windows, prefer building in WSL (make build-artifact) for correct Unix permissions." -ForegroundColor Yellow
}

function Build-SheetSetupArtifact {
    Write-Host "Building sheet-setup Lambda binary into dist/sheet-setup/ (linux/arm64)..." -ForegroundColor Cyan

    if (-not (Test-Path "dist/sheet-setup")) {
        New-Item -ItemType Directory -Path "dist/sheet-setup" -Force | Out-Null
    }

    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    try {
        go build -tags lambda.norpc -o dist/sheet-setup/bootstrap ./cmd/sheet-setup
        if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
    }
    finally {
        Remove-Item Env:\GOOS
        Remove-Item Env:\GOARCH
        Remove-Item Env:\CGO_ENABLED
    }

    Write-Host "Build complete: dist/sheet-setup/bootstrap" -ForegroundColor Green
    Write-Host "Note: For CDK deploy, use: .\deploy.ps1 deploy-cdk -SheetSetupArtifactPath '../../dist/sheet-setup' ..." -ForegroundColor Yellow
    Write-Host "      On Windows, prefer building in WSL (make build-sheet-setup-artifact) for correct Unix permissions." -ForegroundColor Yellow
}

function Build-AllArtifacts {
    Build-Artifact
    Build-SheetSetupArtifact
}

function Create-Lambda {
    if (-not $GoogleSheetId) { throw "GoogleSheetId is required. Use: .\deploy.ps1 create-lambda -GoogleSheetId 'your-id' -AlexaSkillId 'amzn1.ask.skill.xxx'" }
    if (-not $AlexaSkillId) { throw "AlexaSkillId is required. Use: .\deploy.ps1 create-lambda -GoogleSheetId 'your-id' -AlexaSkillId 'amzn1.ask.skill.xxx'" }

    Build-Lambda

    Write-Host "Creating IAM role..." -ForegroundColor Cyan
    $trustPolicy = '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}'
    aws iam create-role `
        --role-name $RoleName `
        --assume-role-policy-document $trustPolicy `
        --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to create IAM role" }

    aws iam attach-role-policy `
        --role-name $RoleName `
        --policy-arn "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
    if ($LASTEXITCODE -ne 0) { throw "Failed to attach execution policy" }

    Write-Host "Waiting for role to propagate..." -ForegroundColor Yellow
    Start-Sleep -Seconds 10

    Write-Host "Creating Lambda function..." -ForegroundColor Cyan
    $roleArn = aws iam get-role --role-name $RoleName --query "Role.Arn" --output text
    if ($LASTEXITCODE -ne 0) { throw "Failed to get role ARN" }

    $envVars = "Variables={GOOGLE_SHEET_ID=$GoogleSheetId,GOOGLE_CREDENTIALS_SECRET=$SecretName,ALEXA_SKILL_ID=$AlexaSkillId,TZ=America/Sao_Paulo}"
    aws lambda create-function `
        --function-name $FunctionName `
        --runtime provided.al2023 `
        --architectures arm64 `
        --handler bootstrap `
        --role $roleArn `
        --zip-file "fileb://function.zip" `
        --environment $envVars `
        --timeout 15 `
        --memory-size 128 `
        --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to create Lambda function" }

    Write-Host "Adding Alexa Skills Kit trigger..." -ForegroundColor Cyan
    aws lambda add-permission `
        --function-name $FunctionName `
        --statement-id alexa-skill-trigger `
        --action "lambda:InvokeFunction" `
        --principal alexa-appkit.amazon.com `
        --event-source-token $AlexaSkillId `
        --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to add Alexa permission" }

    Write-Host "Adding Secrets Manager permission..." -ForegroundColor Cyan
    $smPolicy = @"
{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"secretsmanager:GetSecretValue","Resource":"arn:aws:secretsmanager:${Region}:*:secret:${SecretName}-*"}]}
"@
    aws iam put-role-policy `
        --role-name $RoleName `
        --policy-name SecretsManagerRead `
        --policy-document $smPolicy
    if ($LASTEXITCODE -ne 0) { throw "Failed to add Secrets Manager policy" }

    Remove-Item function.zip -ErrorAction SilentlyContinue

    $arn = aws lambda get-function --function-name $FunctionName --query "Configuration.FunctionArn" --output text --region $Region
    Write-Host "Done! Lambda ARN: $arn" -ForegroundColor Green
}

function Deploy-Lambda {
    Build-Lambda

    Write-Host "Deploying to Lambda..." -ForegroundColor Cyan
    aws lambda update-function-code `
        --function-name $FunctionName `
        --zip-file "fileb://function.zip" `
        --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to update function code" }

    Remove-Item function.zip -ErrorAction SilentlyContinue
    Write-Host "Deploy complete." -ForegroundColor Green
}

function Store-Secret {
    Write-Host "Storing credentials in Secrets Manager..." -ForegroundColor Cyan
    aws secretsmanager create-secret `
        --name $SecretName `
        --description "Google Sheets service account credentials for Alexa attendance skill" `
        --secret-string "file://credentials.json" `
        --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to create secret" }

    Write-Host "Secret stored: $SecretName" -ForegroundColor Green
}

function Build-SheetSetupLambda {
    Write-Host "Building sheet-setup Lambda binary (linux/arm64)..." -ForegroundColor Cyan

    $env:GOOS = "linux"
    $env:GOARCH = "arm64"
    $env:CGO_ENABLED = "0"
    try {
        go build -tags lambda.norpc -o bootstrap ./cmd/sheet-setup
        if ($LASTEXITCODE -ne 0) { throw "Go build failed" }
    }
    finally {
        Remove-Item Env:\GOOS
        Remove-Item Env:\GOARCH
        Remove-Item Env:\CGO_ENABLED
    }

    python -c @"
import zipfile
z = zipfile.ZipFile('sheet-setup.zip', 'w', zipfile.ZIP_DEFLATED)
i = zipfile.ZipInfo('bootstrap')
i.external_attr = 0o755 << 16
z.writestr(i, open('bootstrap', 'rb').read())
z.close()
"@
    if ($LASTEXITCODE -ne 0) { throw "Failed to create sheet-setup.zip with correct permissions" }
    Remove-Item bootstrap

    Write-Host "Build complete: sheet-setup.zip" -ForegroundColor Green
}

function Deploy-Cfn {
    if (-not $GoogleSheetId) { throw "GoogleSheetId is required" }
    if (-not $AlexaSkillId) { throw "AlexaSkillId is required" }
    if (-not $S3Bucket) { throw "S3Bucket is required (for uploading function.zip)" }

    $sheetSetupS3Key = "alexa-skill/sheet-setup.zip"

    Build-Lambda
    Build-SheetSetupLambda

    Write-Host "Uploading function.zip to S3..." -ForegroundColor Cyan
    aws s3 cp function.zip "s3://$S3Bucket/$S3Key" --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to upload to S3" }

    Write-Host "Uploading sheet-setup.zip to S3..." -ForegroundColor Cyan
    aws s3 cp sheet-setup.zip "s3://$S3Bucket/$sheetSetupS3Key" --region $Region
    if ($LASTEXITCODE -ne 0) { throw "Failed to upload to S3" }

    Write-Host "Deploying CloudFormation stack..." -ForegroundColor Cyan
    aws cloudformation deploy `
        --template-file infra/cloudformation/template.yaml `
        --stack-name alexa-attendance-skill `
        --capabilities CAPABILITY_NAMED_IAM `
        --region $Region `
        --parameter-overrides `
            "GoogleSheetId=$GoogleSheetId" `
            "AlexaSkillId=$AlexaSkillId" `
            "GoogleCredentialsSecretName=$SecretName" `
            "LambdaCodeS3Bucket=$S3Bucket" `
            "LambdaCodeS3Key=$S3Key" `
            "SheetSetupCodeS3Key=$sheetSetupS3Key"
    if ($LASTEXITCODE -ne 0) { throw "CloudFormation deploy failed" }

    Remove-Item function.zip, sheet-setup.zip -ErrorAction SilentlyContinue

    $arn = aws cloudformation describe-stacks `
        --stack-name alexa-attendance-skill `
        --query "Stacks[0].Outputs[?OutputKey=='LambdaFunctionArn'].OutputValue" `
        --output text --region $Region
    Write-Host "Done! Lambda ARN: $arn" -ForegroundColor Green
}

function Deploy-Cdk {
    if (-not $GoogleSheetId) { throw "GoogleSheetId is required" }

    Write-Host "Installing CDK dependencies..." -ForegroundColor Cyan
    Push-Location infra/cdk
    try {
        npm install
        if ($LASTEXITCODE -ne 0) { throw "npm install failed" }

        Write-Host "Deploying CDK stack..." -ForegroundColor Cyan
        $cdkArgs = @(
            "cdk", "deploy",
            "--context", "googleSheetId=$GoogleSheetId",
            "--context", "googleCredentialsSecretName=$SecretName",
            "--context", "region=$Region"
        )
        if ($AlexaSkillId) {
            $cdkArgs += "--context", "alexaSkillId=$AlexaSkillId"
        }
        if ($ArtifactPath) {
            $cdkArgs += "--context", "artifactPath=$ArtifactPath"
        }
        if ($SheetSetupArtifactPath) {
            $cdkArgs += "--context", "sheetSetupArtifactPath=$SheetSetupArtifactPath"
        }
        $cdkArgs += "--require-approval", "never"

        npx @cdkArgs
        if ($LASTEXITCODE -ne 0) { throw "CDK deploy failed" }

        Write-Host "CDK deploy complete." -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

# ──────────────────────────────────────────────────
# Command dispatch
# ──────────────────────────────────────────────────

switch ($Command) {
    "build-lambda"    { Build-Lambda }
    "build-artifact"              { Build-Artifact }
    "build-sheet-setup-artifact"  { Build-SheetSetupArtifact }
    "build-all-artifacts"         { Build-AllArtifacts }
    "create-lambda"               { Create-Lambda }
    "deploy-lambda"   { Deploy-Lambda }
    "store-secret"    { Store-Secret }
    "build-server"    { docker compose build }
    "run-server"      { docker compose up }
    "deploy-cfn"      { Deploy-Cfn }
    "deploy-cdk"      { Deploy-Cdk }
}
