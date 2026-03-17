package main

import (
	"alexa-attendance-skill/internal/sheets"
	"context"
	"log/slog"
	"os"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()
	sheetsClient, err := sheets.NewClient(ctx, "", "")
	if err != nil {
		slog.Error("failed to initialize Google Sheets client", "error", err)
		os.Exit(1)
	}

	if err := sheetsClient.EnsureHeaders(ctx); err != nil {
		slog.Error("EnsureHeaders failed", "error", err)
		os.Exit(1)
	}

	if err := sheetsClient.RecordArrival(ctx, "Test"); err != nil {
		slog.Error("RecordArrival failed", "error", err)
		os.Exit(1)
	}
}
