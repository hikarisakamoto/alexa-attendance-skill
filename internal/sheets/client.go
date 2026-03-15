package sheets

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/api/option"
	sheetsv4 "google.golang.org/api/sheets/v4"
)

const (
	colName    = "Nome"
	colArrival = "Chegada"
	colLeave   = "Saída"
)

var brTimeZone = func() *time.Location {
	location, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		slog.Warn("failed to load timezone, falling back to UTC", "error", err)
		return time.UTC
	}
	return location
}()

func todaySheetName() string {
	return time.Now().In(brTimeZone).Format("2006-01-02")
}

type sheetsAPI interface {
	batchUpdate(ctx context.Context, sheetID string, req *sheetsv4.BactchUpdateSpreadsheetRequest) error
	getValues(ctx context.Context, sheetID string, cellRange string) (*sheetsv4.ValueRange, error)
	updateValues(ctx context.Context, sheetID string, cellRange string, values *sheetsv4.ValueRange) error
	appendValues(ctx context.Context, sheetID string, cellRange string, values *sheetsv4.ValueRange) error
}

type googleSheetsAPI struct{ sheetService *sheetsv4.Service }

func (s *googleSheetsAPI) batchUpdate(ctx context.Context, sheetID string, req *sheetsv4.BactchUpdateSpreadsheetRequest) error {
	_, err := s.sheetService.Spreadsheets.BatchUpdate(sheetID, req).Context(ctx).Do()
	return err
}

func (s *googleSheetsAPI) getValues(ctx context.Context, sheetID, cellRange string) (*sheetsv4.ValueRange, error) {
	return s.sheetService.Spreadsheets.Values.Get(sheetID, cellRange).Context(ctx).Do()
}

func (s *googleSheetsAPI) updateValues(ctx context.Context, sheetID, cellRange string, values *sheetsv4.ValueRange) error {
	_, err := s.sheetService.Spreadsheets.Values.Update(sheetID, cellRange, values).ValueInputOption("RAW").Context(ctx).Do()
	return err
}

func (s *googleSheetsAPI) appendValues(ctx context.Context, sheetID, cellRange string, values *sheetsv4.ValueRange) error {
	_, err := s.sheetService.Spreadsheets.Values.Append(sheetID, cellRange, values).ValueInputOption("RAW").InsertDataOption("INSERT_ROWS").Context(ctx).Do()
	return err
}

type Client struct {
	api              sheetsAPI
	sheetID          string
	ensuredSheetName string
	mu               sync.RWMutex
}

func NewClient(ctx context.Context, credentials string, sheetID string) (*Client, error) {
	slog.Info("creating sheets client from file", "sheetID", sheetID)
	service, err := sheetsv4.NewService(ctx, option.WithAuthCredentialsFile(option.ServiceAccount, credentials))
	if err != nil {
		return nil, fmt.Errorf("unable to create sheets service: %w", err)
	}
	slog.Info("sheets client created successfully")
	return &Client{api: &googleSheetsAPI{sheetService: service}, sheetID: sheetID}, nil
}

func NewClientFromJSON(ctx context.Context, credentialsJSON []byte, sheetID string) (*Client, error) {
	slog.Info("creating sheets client from JSON", "sheetID", sheetID)
	service, err := sheetsv4.NewService(ctx, option.WithAuthCredentialsJSON(option.ServiceAccount, credentialsJSON))
	if err != nil {
		return nil, fmt.Errorf("unable to create sheets service %w", &err)
	}
	slog.Info("sheets client created successfully")
	return &Client{api: &googleSheetsAPI{sheetService: service}, sheetID: sheetID}, nil
}
