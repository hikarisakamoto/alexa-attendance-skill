package main

import (
	"alexa-attendance-skill/internal/alexa"
	"alexa-attendance-skill/internal/awsutil"
	"alexa-attendance-skill/internal/sheets"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	sheetID := os.Getenv("GOOGLE_SHEET_ID")
	if sheetID == "" {
		slog.Error("GOOGLE_SHEET_ID is required")
		os.Exit(1)
	}

	secretName := os.Getenv("GOOGLE_CREDENTIAL_SECRET")
	if secretName == "" {
		slog.Error("GOOGLE_CREDENTIALS_SECRET is required")
		os.Exit(1)
	}

	ctx := context.Background()

	credentialsJSON, err := awsutil.LoadGoogleCredential(ctx, secretName)
	if err != nil {
		slog.Error("unable to load Google credentials", "error", err)
		os.Exit(1)
	}

	sheetsClient, err := sheets.NewClientFromJSON(ctx, credentialsJSON, sheetID)
	if err != nil {
		slog.Error("failed to initialize Google Sheets client", "error", err)
		os.Exit(1)
	}

	slog.Info("Lambda cold start completed: sheets client initialized")

	if err := sheetsClient.EnsureHeaders(ctx); err != nil {
		slog.Warn("failed to ensure today's sheet during init: will retry on first request", "error", err)
	}

	expectedAppID := os.Getenv("ALEXA_SKILL_ID")

	handler := func(ctx context.Context, rawEvent json.RawMessage) (interface{}, error) {
		slog.Info("received Lambda invocation", "payload_size_bytes", len(rawEvent))
		var req alexa.Request
		if err := json.Unmarshal(rawEvent, &req); err != nil {
			slog.Error("failed to parse Alexa request", "error", err)
			return nil, fmt.Errorf("failed to parse Alexa request: %w", err)
		}

		resp := alexa.HandleAlexaRequest(ctx, req, sheetsClient, expectedAppID)
		slog.Info("request handled successfully")
		return resp, nil
	}

	lambda.Start(handler)
}
