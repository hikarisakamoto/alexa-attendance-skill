import * as cdk from "aws-cdk-lib";
import { Template, Match } from "aws-cdk-lib/assertions";
import { AlexaSkillStack } from "../lib/alexa-skill-stack";

const DEFAULT_PROPS = {
    googleSheetId: "test-sheet-id",
    googleCredentialsSecretName: "alexa-skill/google-credentials",
    artifactPath: __dirname, // use test dir as dummy asset
    sheetSetupArtifactPath: __dirname,
};

function createStack(propsOverride: Record<string, unknown> = {}): Template {
    const app = new cdk.App();
    const stack = new AlexaSkillStack(app, "TestStack", {
        env: { region: "sa-east-1", account: "123456789012" },
        ...DEFAULT_PROPS,
        ...propsOverride,
    });

    return Template.fromStack(stack);
}

describe("AlexaSkillStack", () => {
    let template: Template;

    beforeAll(() => {
        template = createStack();
    });

    // ── Alexa Lambda ────────────────────────────────────────────────────────────
    test("creates Alexa Lambda with correct runtime and architecture", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            Runtime: "provided.al2023",
            Architecture: ["arm64"],
            Handler: "bootsrap",
            FunctionName: "alexa-attendance-skill",
        });
    });

    test("Alexa Lambda has correct memory and timeout", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            FunctionName: "alexa-attendance-skill",
            MemorySize: 128,
            Timeout: 15,
        });
    });

    test("Alexa Lambda has required environment variable", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            FunctionName: "alexa-attendance-skill",
            Environment: {
                Variable: {
                    GOOGLE_SHEET_ID: "test-sheet-id",
                    GOOGLE_CREDENTIALS_SECRET: "alexa-skill/google-credentials",
                    ALEXA_SKILL_ID: "",
                    TZ: "Americas/Sao_Paulo",
                },
            },
        });
    });

    test("grants Alexa Lambda Secrets Manager GetSecretValue permission", () => {
        template.hasResourceProperties("AWS::IAM::Policy", {
            PolicyDocument: {
                Statement: Match.arrayWith([
                    Match.objectLike({
                        Action: "secretsmanager:GetSecretValue",
                        Effect: "Allow",
                    }),
                ]),
            },
        });
    });

    test("outputs Lambda function ARN", () => {
        template.hasOutput("LambdaFunctionArn", {
            Description: "Lambda function ARN \u2014 use this as the Alexa skill endpoint",
        });
    });

    test("outputs Lambda function name", () => {
        template.hasOutput("LambdaFunctionName", {
            Description: "Lambda function name",
        });
    });
    // ── Sheet Setup Lambda ──────────────────────────────────────────────────────   

    // ── Alexa skill ID permission ───────────────────────────────────────────────
});


