package main

import (
	"alexa-attendance-skill/internal/awsutil"
	"alexa-attendance-skill/internal/sheets"
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	sheetID := os.Getenv("GOOGLE_SHEED_ID")
	if sheetID == "" {
		slog.Error("GOOGLE_SHEET_ID is required")
		os.Exit(1)
	}

	secretName := os.Getenv("GOOGLE_CREDENTIALS_SECERT")
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

	slog.Info("sheet-setup cold start completed: sheets client initialized")

	handler := func(ctx context.Context) error {
		slog.Info("sheet-setup invoked: ensuring today's sheet tab exists")
		if err := sheetsClient.EnsureHeaders(ctx); err != nil {
			slog.Error("sheet-setup failed", "error", err)
			return err
		}
		slog.Info("sheet-setup completed successfully")
		return nil
	}

	lambda.Start(handler)
}
