import * as cdk from "aws-cdk-lib";
import * as lambda from "aws-cdk-lib/aws-lambda";
import * as logs from "aws-cdk-lib/aws-logs";
import * as iam from "aws-cdk-lib/aws-iam";
import * as events from "aws-cdk-lib/aws-events";
import * as eventsTargets from "aws-cdk-lib/aws-events-targets";
import { Construct } from "constructs";
import * as path from "path";
import { execSync } from "child_process";

interface AlexaSkillStackProps extends cdk.StackProps {
    googleSheetId: string
    alexaSkillId?: string
    googleCredentialsSecretName: string
    artifactPath?: string
    sheetSetupArtifactPath?: string
}

export class AlexaSkillStack extends cdk.Stack {
    constructor(scope: Construct, id: string, props: AlexaSkillStackProps) {
        super(scope, id, props);

        const projectRoot = path.join(__dirname, "..", "..", "..", "..");
        const artifactPath = props.artifactPath ? path.resolve(props.artifactPath) : undefined;
        const sheetSetupArtifactPath = props.sheetSetupArtifactPath
            ? path.resolve(props.sheetSetupArtifactPath)
            : undefined;

        // ── Alexa Lambda ──────────────────────────────────────────────────────────

        const logGroup = new logs.LogGroup(this, "AlexaSkillLogGroup", {
            logGroupName: "aws/lambda/alexa-attendance-skill",
            retention: logs.RetentionDays.TWO_WEEKS,
            removalPolicy: cdk.RemovalPolicy.RETAIN
        });

        const fn = new lambda.Function(this, "AlexaSkillFunction", {
            functionName: "alexa-attendance-skill",
            runtime: lambda.Runtime.PROVIDED_AL2023,
            architecture: lambda.Architecture.ARM_64,
            handler: "bootstrap",
            code: artifactPath
                ? lambda.Code.fromAsset(artifactPath)
                : lambda.Code.fromAsset(projectRoot, {
                    bundling: {
                        image: cdk.DockerImage.fromRegistry("public.ecr.aws/sam/build-provided.al2023"),
                        local: {
                            tryBundle(outputDir: string) {
                                execSync(
                                    `go build -tags lambda.norpc -o "${path.join(outputDir, "bootstrap")}" ./cmd/lambda`,
                                    {
                                        cwd: projectRoot,
                                        stdio: "inherit",
                                        env: {
                                            ...process.env,
                                            CGO_ENABLED: "0",
                                            GOOS: "linux",
                                            GOARCH: "arm64",
                                        },
                                    }
                                );
                                return true;
                            },
                        },
                    },
                }),
            memorySize: 128,
            timeout: cdk.Duration.seconds(15),
            logGroup,
            environment: {
                GOOGLE_SHEET_ID: props.googleSheetId,
                GOOGLE_CREDENTIALS_SECRET: props.googleCredentialsSecretName,
                ALEXA_SKILL_ID: props.alexaSkillId ?? "",
                TZ: "America/Sao_Paulo"
            },
        });

        fn.addToRolePolicy(
            new iam.PolicyStatement({
                actions: ["secretsmanager:GetSecretValue"],
                resources: [
                    `arn:aws:secretsmanager:${this.region}:${this.account}:secret:${props.googleCredentialsSecretName}-*`,
                ],
            })
        );

        if (props.alexaSkillId) {
            fn.addPermission("AlexaSkillInvoke", {
                principal: new iam.ServicePrincipal("alexa-appkit.amazon.com"),
                action: "lambda:InvokeFunction",
                eventSourceToken: props.alexaSkillId,
            });
        }

        // ── Sheet Setup Lambda (scheduled daily) ─────────────────────────────────

        const sheetSetupLogGroup = new logs.LogGroup(this, "SheetSetupLogGroup", {
            logGroupName: "aws/lambda/alexa-sheet-setup",
            retention: logs.RetentionDays.TWO_WEEKS,
            removalPolicy: cdk.RemovalPolicy.RETAIN
        });

        const sheetSetupFn = new lambda.Function(this, "SheetSetupFunction", {
            functionName: "alexa-sheet-setup",
            runtime: lambda.Runtime.PROVIDED_AL2023,
            architecture: lambda.Architecture.ARM_64,
            handler: "bootstrap",
            code: sheetSetupArtifactPath
                ? lambda.Code.fromAsset(sheetSetupArtifactPath)
                : lambda.Code.fromAsset(projectRoot, {
                    bundling: {
                        image: cdk.DockerImage.fromRegistry("public.ecr.aws/sam/build-provided.al2023"),
                        local: {
                            tryBundle(outputDir: string) {
                                execSync(
                                    `go build -tags lambda.norpc -o "${path.join(outputDir, "bootstrap")}" ./cmd/sheet-setup`,
                                    {
                                        cwd: projectRoot,
                                        stdio: "inherit",
                                        env: {
                                            ...process.env,
                                            CGO_ENABLED: "0",
                                            GOOS: "linux",
                                            GOARCH: "arm64",
                                        },
                                    }
                                );
                                return true;
                            },
                        },
                    },
                }),
            memorySize: 128,
            timeout: cdk.Duration.seconds(30),
            logGroup: sheetSetupLogGroup,
            environment: {
                GOOGLE_SHEET_ID: props.googleSheetId,
                GOOGLE_CREDENTIALS_SECRET: props.googleCredentialsSecretName,
                TZ: "America/Sao_Paulo"
            },
        });

        sheetSetupFn.addToRolePolicy(
            new iam.PolicyStatement({
                actions: ["secretsmanager:GetSecretValue"],
                resources: [
                    `arn:aws:secretsmanager:${this.region}:${this.account}:secret:${props.googleCredentialsSecretName}-*`,
                ],
            })
        );

        // EventBridge rule: 03:00 UTC = midnight São Paulo time
        const scheduleRule = new events.Rule(this, "SheetSetupScheduleRule", {
            ruleName: "alexa-sheet-setup-daily",
            description: "Creates the daily Google Sheets tab at midnight São Paulo time",
            schedule: events.Schedule.cron({ minute: "0", hour: "3" }), // 03:00 UTC
        });

        scheduleRule.addTarget(new eventsTargets.LambdaFunction(sheetSetupFn));

        // ── Outputs ───────────────────────────────────────────────────────────────
        new cdk.CfnOutput(this, "LambdaFunctionArn", {
            description: "Lambda function ARN \u2014 use this as the Alexa skill endpoint",
            value: fn.functionArn,
        });

        new cdk.CfnOutput(this, "LambdaFunctionName", {
            description: "Lambda functino name",
            value: fn.functionName,
        });
    }
}