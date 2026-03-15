package sheets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/googleapi"
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
	batchUpdate(ctx context.Context, sheetID string, req *sheetsv4.BatchUpdateSpreadsheetRequest) error
	getValues(ctx context.Context, sheetID string, cellRange string) (*sheetsv4.ValueRange, error)
	updateValues(ctx context.Context, sheetID string, cellRange string, values *sheetsv4.ValueRange) error
	appendValues(ctx context.Context, sheetID string, cellRange string, values *sheetsv4.ValueRange) error
}

type googleSheetsAPI struct{ sheetService *sheetsv4.Service }

func (s *googleSheetsAPI) batchUpdate(ctx context.Context, sheetID string, req *sheetsv4.BatchUpdateSpreadsheetRequest) error {
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

func (c *Client) EnsureSheet(ctx context.Context, name string) error {
	slog.Info("ensuring sheet tab exists", "tabName", name)
	req := &sheetsv4.BatchUpdateSpreadsheetRequest{
		Requests: []*sheetsv4.Request{
			{
				AddSheet: &sheetsv4.AddSheetRequest{
					Properties: &sheetsv4.SheetProperties{
						Title: name,
					},
				},
			},
		},
	}

	err := c.api.batchUpdate(ctx, c.sheetID, req)
	if err != nil {
		var googleErr *googleapi.Error
		if errors.As(err, *googleErr) && googleErr.Code == 400 && strings.Contains(googleErr.Message, "already exists") {
			slog.Info("sheet tab already exists", "tabName", name)
			return nil
		}
		if strings.Contains(err.Error(), "already exists") {
			slog.Info("sheet tab already exists", "tabName", name)
			return nil
		}
		return fmt.Errorf("unable to create sheet %q: %w", name, err)
	}
	slog.Info("sheet tab created", "tabName", name)
	return nil
}

func (c *Client) EnsureHeaders(ctx context.Context) error {
	name := todaySheetName()

	c.mu.RLock()
	cachedName := c.ensuredSheetName
	c.mu.RUnlock()

	if cachedName == name {
		slog.Info("sheet already ensured, skipping", "tabName", name)
		return nil
	}

	if err := c.EnsureSheet(ctx, name); err != nil {
		return err
	}

	slog.Info("checking headers on sheet", "tabName", name)
	readRange := fmt.Sprintf("'%s'!A1:C1", name)
	resp, err := c.api.getValues(ctx, c.sheetID, readRange)
	if err != nil {
		return fmt.Errorf("unable to read sheet: %w", err)
	}

	if len(resp.Values) > 0 && len(resp.Values[0]) >= 3 {
		slog.Info("headers already present on sheet", "tabName", name)
		c.mu.Lock()
		c.ensuredSheetName = name
		c.mu.Unlock()
		return nil
	}

	slog.Info("writing headers to sheet", "tabName", name)
	headers := &sheetsv4.ValueRange{
		Values: [][]interface{}{{colName, colArrival, colLeave}},
	}
	writeRange := fmt.Sprintf("'%s'!A1:C1", name)
	if err := c.api.updateValues(ctx, c.sheetID, writeRange, headers); err != nil {
		return fmt.Errorf("unable to write headers: %w", err)
	}
	slog.Info("headers written to sheet", "tabName", name)
	c.mu.Lock()
	c.ensuredSheetName = name
	c.mu.Unlock()
	return nil
}
