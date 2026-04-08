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

    test("creates SheetSetup Lambda with correct runtime and architecture", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            Runtime: "provided.al2023",
            Architecture: ["arm64"],
            Handler: "bootstrap",
            FunctionName: "alexa-sheet-setup",
        });
    });

    test("SheetSetup Lambda has correct memory and timeout", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            FunctionName: "alexa-sheet-setup",
            MemorySize: 128,
            Timeout: 30,
        });
    });

    test("SheetSetup Lambda has required environment variable", () => {
        template.hasResourceProperties("AWS::Lambda::Function", {
            FunctionName: "alexa-sheet-setup",
            Environment: {
                Variables: {
                    GOOGLE_SHEET_ID: "test-sheet-id",
                    GOOGLE_CREDENTIALS_SECRET: "alexa-skill/google-credentials",
                    TZ: "America/Sao_Paulo",
                },
            },
        });
    });

    test("creates EventBridge rule targeting SheetSetup Lambda", () => {
        template.hasResourceProperties("AWS::Events::Rule", {
            Name: "alexa-sheet-setup-daily",
            SheduleExpression: "cron(0 3 * * ? *)",
            State: "ENABLED",
        });
    });

    // ── Alexa skill ID permission ───────────────────────────────────────────────

    describe("when alexaSkillId is provided", () => {
        let templateWithSkillId: Template;

        beforeAll(() => {
            templateWithSkillId = createStack({
                alexaSkillId: "amzn1.ask.skill.test-id",
            });
        });

        test("adds Alexa invoke permission", () => {
            templateWithSkillId.hasResourceProperties("AWS::Lambda::Permission", {
                Action: "lambda:InvokeFunction",
                Principal: "alexa-appkit.amazon.com",
                EventSourceToken: "amzn1.ask.skill.test-id",
            });
        });

        test("sets ALEXA_SKILL_ID env var on Alexa Lambda", () => {
            templateWithSkillId.hasResourceProperties("AWS::Lambda::Function", {
                FunctionName: "alexa-attendance-skill",
                Environment: {
                    Variables: Match.objectLike({
                        ALEXA_SKILL_ID: "amzn1.ask.skill.test-id",
                    }),
                },
            });
        });
    });

    describe("when alexaSkillId is NOT provide", () => {

        test("does not create an Alexa Lambda permission", () => {
            // Only the EventBridge -> SheetSetup permisson should exist
            template.hasResourceProperties("AWS::Lambda::Permission", {
                Principal: "events.amazonaws.com",
            });
            template.resourceCountIs("AWS::Lambda::Permission", 1);
        });
    });

    test("matches snapshot", () => {
        expect(template.toJSON()).toMatchInlineSnapshot(`
{
  "Outputs": {
    "LambdaFunctionArn": {
      "Description": "Lambda function ARN — use this as the Alexa skill endpoint",
      "Value": {
        "Fn::GetAtt": [
          "AlexaSkillFunctionE4AF1F33",
          "Arn",
        ],
      },
    },
    "LambdaFunctionName": {
      "Description": "Lambda functino name",
      "Value": {
        "Ref": "AlexaSkillFunctionE4AF1F33",
      },
    },
  },
  "Parameters": {
    "BootstrapVersion": {
      "Default": "/cdk-bootstrap/hnb659fds/version",
      "Description": "Version of the CDK Bootstrap resources in this environment, automatically retrieved from SSM Parameter Store. [cdk:skip]",
      "Type": "AWS::SSM::Parameter::Value<String>",
    },
  },
  "Resources": {
    "AlexaSkillFunctionE4AF1F33": {
      "DependsOn": [
        "AlexaSkillFunctionServiceRoleDefaultPolicyDC5D4689",
        "AlexaSkillFunctionServiceRole753B5A53",
      ],
      "Properties": {
        "Architectures": [
          "arm64",
        ],
        "Code": {
          "S3Bucket": "cdk-hnb659fds-assets-123456789012-sa-east-1",
          "S3Key": "fdec0340d2a998a2f81db3dc47ddedbe2027dd95ae69f834b8f7ec28e837e534.zip",
        },
        "Environment": {
          "Variables": {
            "ALEXA_SKILL_ID": "",
            "GOOGLE_CREDENTIALS_SECRET": "alexa-skill/google-credentials",
            "GOOGLE_SHEET_ID": "test-sheet-id",
            "TZ": "America/Sao_Paulo",
          },
        },
        "FunctionName": "alexa-attendance-skill",
        "Handler": "bootstrap",
        "LoggingConfig": {
          "LogGroup": {
            "Ref": "AlexaSkillLogGroup10463C26",
          },
        },
        "MemorySize": 128,
        "Role": {
          "Fn::GetAtt": [
            "AlexaSkillFunctionServiceRole753B5A53",
            "Arn",
          ],
        },
        "Runtime": "provided.al2023",
        "Timeout": 15,
      },
      "Type": "AWS::Lambda::Function",
    },
    "AlexaSkillFunctionServiceRole753B5A53": {
      "Properties": {
        "AssumeRolePolicyDocument": {
          "Statement": [
            {
              "Action": "sts:AssumeRole",
              "Effect": "Allow",
              "Principal": {
                "Service": "lambda.amazonaws.com",
              },
            },
          ],
          "Version": "2012-10-17",
        },
        "ManagedPolicyArns": [
          {
            "Fn::Join": [
              "",
              [
                "arn:",
                {
                  "Ref": "AWS::Partition",
                },
                ":iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
              ],
            ],
          },
        ],
      },
      "Type": "AWS::IAM::Role",
    },
    "AlexaSkillFunctionServiceRoleDefaultPolicyDC5D4689": {
      "Properties": {
        "PolicyDocument": {
          "Statement": [
            {
              "Action": "secretsmanager:GetSecretValue",
              "Effect": "Allow",
              "Resource": "arn:aws:secretsmanager:sa-east-1:123456789012:secret:alexa-skill/google-credentials-*",
            },
          ],
          "Version": "2012-10-17",
        },
        "PolicyName": "AlexaSkillFunctionServiceRoleDefaultPolicyDC5D4689",
        "Roles": [
          {
            "Ref": "AlexaSkillFunctionServiceRole753B5A53",
          },
        ],
      },
      "Type": "AWS::IAM::Policy",
    },
    "AlexaSkillLogGroup10463C26": {
      "DeletionPolicy": "Retain",
      "Properties": {
        "LogGroupName": "aws/lambda/alexa-attendance-skill",
        "RetentionInDays": 14,
      },
      "Type": "AWS::Logs::LogGroup",
      "UpdateReplacePolicy": "Retain",
    },
    "SheetSetupFunction49925CE5": {
      "DependsOn": [
        "SheetSetupFunctionServiceRoleDefaultPolicy20DCE534",
        "SheetSetupFunctionServiceRole31809E1F",
      ],
      "Properties": {
        "Architectures": [
          "arm64",
        ],
        "Code": {
          "S3Bucket": "cdk-hnb659fds-assets-123456789012-sa-east-1",
          "S3Key": "fdec0340d2a998a2f81db3dc47ddedbe2027dd95ae69f834b8f7ec28e837e534.zip",
        },
        "Environment": {
          "Variables": {
            "GOOGLE_CREDENTIALS_SECRET": "alexa-skill/google-credentials",
            "GOOGLE_SHEET_ID": "test-sheet-id",
            "TZ": "America/Sao_Paulo",
          },
        },
        "FunctionName": "alexa-sheet-setup",
        "Handler": "bootstrap",
        "LoggingConfig": {
          "LogGroup": {
            "Ref": "SheetSetupLogGroupB3F9CB6A",
          },
        },
        "MemorySize": 128,
        "Role": {
          "Fn::GetAtt": [
            "SheetSetupFunctionServiceRole31809E1F",
            "Arn",
          ],
        },
        "Runtime": "provided.al2023",
        "Timeout": 30,
      },
      "Type": "AWS::Lambda::Function",
    },
    "SheetSetupFunctionServiceRole31809E1F": {
      "Properties": {
        "AssumeRolePolicyDocument": {
          "Statement": [
            {
              "Action": "sts:AssumeRole",
              "Effect": "Allow",
              "Principal": {
                "Service": "lambda.amazonaws.com",
              },
            },
          ],
          "Version": "2012-10-17",
        },
        "ManagedPolicyArns": [
          {
            "Fn::Join": [
              "",
              [
                "arn:",
                {
                  "Ref": "AWS::Partition",
                },
                ":iam::aws:policy/service-role/AWSLambdaBasicExecutionRole",
              ],
            ],
          },
        ],
      },
      "Type": "AWS::IAM::Role",
    },
    "SheetSetupFunctionServiceRoleDefaultPolicy20DCE534": {
      "Properties": {
        "PolicyDocument": {
          "Statement": [
            {
              "Action": "secretsmanager:GetSecretValue",
              "Effect": "Allow",
              "Resource": "arn:aws:secretsmanager:sa-east-1:123456789012:secret:alexa-skill/google-credentials-*",
            },
          ],
          "Version": "2012-10-17",
        },
        "PolicyName": "SheetSetupFunctionServiceRoleDefaultPolicy20DCE534",
        "Roles": [
          {
            "Ref": "SheetSetupFunctionServiceRole31809E1F",
          },
        ],
      },
      "Type": "AWS::IAM::Policy",
    },
    "SheetSetupLogGroupB3F9CB6A": {
      "DeletionPolicy": "Retain",
      "Properties": {
        "LogGroupName": "aws/lambda/alexa-sheet-setup",
        "RetentionInDays": 14,
      },
      "Type": "AWS::Logs::LogGroup",
      "UpdateReplacePolicy": "Retain",
    },
    "SheetSetupScheduleRuleAllowEventRuleTestStackSheetSetupFunction02C8CA4AC98D4859": {
      "Properties": {
        "Action": "lambda:InvokeFunction",
        "FunctionName": {
          "Fn::GetAtt": [
            "SheetSetupFunction49925CE5",
            "Arn",
          ],
        },
        "Principal": "events.amazonaws.com",
        "SourceArn": {
          "Fn::GetAtt": [
            "SheetSetupScheduleRuleD9F2D1C4",
            "Arn",
          ],
        },
      },
      "Type": "AWS::Lambda::Permission",
    },
    "SheetSetupScheduleRuleD9F2D1C4": {
      "Properties": {
        "Description": "Creates the daily Google Sheets tab at midnight São Paulo time",
        "Name": "alexa-sheet-setup-daily",
        "ScheduleExpression": "cron(0 3 * * ? *)",
        "State": "ENABLED",
        "Targets": [
          {
            "Arn": {
              "Fn::GetAtt": [
                "SheetSetupFunction49925CE5",
                "Arn",
              ],
            },
            "Id": "Target0",
          },
        ],
      },
      "Type": "AWS::Events::Rule",
    },
  },
  "Rules": {
    "CheckBootstrapVersion": {
      "Assertions": [
        {
          "Assert": {
            "Fn::Not": [
              {
                "Fn::Contains": [
                  [
                    "1",
                    "2",
                    "3",
                    "4",
                    "5",
                  ],
                  {
                    "Ref": "BootstrapVersion",
                  },
                ],
              },
            ],
          },
          "AssertDescription": "CDK bootstrap stack version 6 required. Please run 'cdk bootstrap' with a recent version of the CDK CLI.",
        },
      ],
    },
  },
}
`);
    });
});


