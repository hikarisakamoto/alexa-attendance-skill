#!/usr/bin/env node
import * as cdk from "aws-cdk-lib";
import { AlexaSkillStack } from "../lib/alexa-skill-stack";

const app = new cdk.App();

const googleSheetId = app.node.tryGetContext("googleSheetId");
if (!googleSheetId) {
    throw new Error("Missing required context: googleSheetId. Pass it with --context googleSheetId=MY_SHEET_ID");
}

new AlexaSkillStack(app, "AlexaAttendanceSkill", {
    env: {
        region: app.node.tryGetContext("region") || "sa-east-1",
    },
    googleSheetId,
    alexaSkillId: app.node.tryGetContext("alexaSkillId"),
    googleCredentialsSecretName: app.node.tryGetContext("googleCredentialsSecretName") || "alexa-skill/google-credentials",
    artifactPath: app.node.tryGetContext("artifactPath"),
    sheetSetupArtifactPath: app.node.tryGetContext("sheetSetupArtifactPath"),
});