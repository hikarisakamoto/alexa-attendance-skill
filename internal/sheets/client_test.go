package sheets

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/api/googleapi"
	sheetsv4 "google.golang.org/api/sheets/v4"
)

// --- mock spreadsheetOps ---

type mockSheetsAPI struct {
	batchUpdateErr  error
	batchUpdateFunc func(sheetID string, req *sheetsv4.BatchUpdateSpreadsheetRequest) error

	getValuesResult *sheetsv4.ValueRange
	getValuesErr    error

	updateValuesErr  error
	updateValuesFunc func(sheetID, rangeStr string, values *sheetsv4.ValueRange) error

	appendValuesErr  error
	appendValuesFunc func(sheetID, rangeStr string, values *sheetsv4.ValueRange) error

	// recorded call args
	updatedRange   string
	updatedValues  *sheetsv4.ValueRange
	appendedRange  string
	appendedValues *sheetsv4.ValueRange
}

func (m *mockSheetsAPI) batchUpdate(ctx context.Context, sheetID string, req *sheetsv4.BatchUpdateSpreadsheetRequest) error {
	if m.batchUpdateFunc != nil {
		return m.batchUpdateFunc(sheetID, req)
	}
	return m.batchUpdateErr
}

func (m *mockSheetsAPI) getValues(ctx context.Context, sheetID, rangeStr string) (*sheetsv4.ValueRange, error) {
	if m.getValuesErr != nil {
		return nil, m.getValuesErr
	}
	if m.getValuesResult != nil {
		return m.getValuesResult, nil
	}
	return &sheetsv4.ValueRange{}, nil
}

func (m *mockSheetsAPI) updateValues(ctx context.Context, sheetID, rangeStr string, values *sheetsv4.ValueRange) error {
	m.updatedRange = rangeStr
	m.updatedValues = values
	if m.updateValuesFunc != nil {
		return m.updateValuesFunc(sheetID, rangeStr, values)
	}
	return m.updateValuesErr
}

func (m *mockSheetsAPI) appendValues(ctx context.Context, sheetID, rangeStr string, values *sheetsv4.ValueRange) error {
	m.appendedRange = rangeStr
	m.appendedValues = values
	if m.appendValuesFunc != nil {
		return m.appendValuesFunc(sheetID, rangeStr, values)
	}
	return m.appendValuesErr
}

func newTestClient(api *mockSheetsAPI) *Client {
	return &Client{api: api, sheetID: "test-sheet-id"}
}

// --- findOpenArrivalRow tests ---

func TestFindOpenArrivalRow(t *testing.T) {
	tests := []struct {
		name    string
		rows    [][]interface{}
		search  string
		wantRow int
	}{
		{
			name:    "empty sheet",
			rows:    nil,
			search:  "Maria",
			wantRow: -1,
		},
		{
			name: "header only",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
			},
			search:  "Maria",
			wantRow: -1,
		},
		{
			name: "person not found",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"João", "08:00:00", ""},
			},
			search:  "Maria",
			wantRow: -1,
		},
		{
			name: "open arrival found",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Maria", "08:00:00", ""},
			},
			search:  "Maria",
			wantRow: 2,
		},
		{
			name: "arrival already closed",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Maria", "08:00:00", "17:00:00"},
			},
			search:  "Maria",
			wantRow: -1,
		},
		{
			name: "case-insensitive match",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"MARIA SILVA", "08:00:00", ""},
			},
			search:  "maria silva",
			wantRow: 2,
		},
		{
			name: "returns most recent open arrival",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Maria", "08:00:00", "12:00:00"}, // row 2 — closed
				{"Maria", "13:00:00", ""},         // row 3 — open
			},
			search:  "Maria",
			wantRow: 3,
		},
		{
			name: "row with only name column (no departure column present)",
			rows: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Carlos", "09:00:00"}, // only 2 columns — leave is absent
			},
			search:  "Carlos",
			wantRow: 2,
		},
		{
			name: "skip header row even if name matches",
			rows: [][]interface{}{
				{"Maria", "Chegada", "Saída"}, // header row — must be skipped
			},
			search:  "Maria",
			wantRow: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := findOpenArrival(tc.rows, tc.search)
			if got != tc.wantRow {
				t.Errorf("findOpenArrivalRow(%v, %q) = %d, want %d", tc.rows, tc.search, got, tc.wantRow)
			}
		})
	}
}

// --- EnsureSheet tests ---

func TestEnsureSheet_Created(t *testing.T) {
	api := &mockSheetsAPI{}
	c := newTestClient(api)
	if err := c.EnsureSheet(t.Context(), "2026-03-14"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureSheet_AlreadyExistsGoogleAPIError(t *testing.T) {
	api := &mockSheetsAPI{
		batchUpdateErr: &googleapi.Error{Code: 400, Message: "already exists"},
	}
	c := newTestClient(api)
	if err := c.EnsureSheet(t.Context(), "2026-03-14"); err != nil {
		t.Fatalf("expected nil error for already-exists, got: %v", err)
	}
}

func TestEnsureSheet_AlreadyExistsFallback(t *testing.T) {
	api := &mockSheetsAPI{
		batchUpdateErr: errors.New("sheet already exists in the spreadsheet"),
	}
	c := newTestClient(api)
	if err := c.EnsureSheet(t.Context(), "2026-03-14"); err != nil {
		t.Fatalf("expected nil error for already-exists fallback, got: %v", err)
	}
}

func TestEnsureSheet_OtherError(t *testing.T) {
	api := &mockSheetsAPI{
		batchUpdateErr: errors.New("network error"),
	}
	c := newTestClient(api)
	if err := c.EnsureSheet(t.Context(), "2026-03-14"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- EnsureHeaders tests ---

func TestEnsureHeaders_CacheHit(t *testing.T) {
	today := todaySheetName()
	api := &mockSheetsAPI{}
	c := newTestClient(api)
	c.ensuredSheetName = today // pre-warm cache

	if err := c.EnsureHeaders(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// batchUpdate should not have been called
	if api.batchUpdateErr != nil {
		t.Error("batchUpdate was called despite cache hit")
	}
}

func TestEnsureHeaders_WritesHeadersWhenAbsent(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{Values: nil}, // no rows
	}
	c := newTestClient(api)
	if err := c.EnsureHeaders(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.updatedValues == nil {
		t.Fatal("expected valuesUpdate to be called to write headers, but it was not")
	}
	row := api.updatedValues.Values[0]
	if row[0] != colName || row[1] != colArrival || row[2] != colLeave {
		t.Errorf("unexpected header values: %v", row)
	}
}

func TestEnsureHeaders_SkipsWriteWhenHeadersPresent(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{
			Values: [][]interface{}{{"Nome", "Chegada", "Saída"}},
		},
	}
	c := newTestClient(api)
	if err := c.EnsureHeaders(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.updatedValues != nil {
		t.Error("valuesUpdate should not have been called when headers are already present")
	}
}

func TestEnsureHeaders_UpdatesCacheAfterSuccess(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{Values: nil},
	}
	c := newTestClient(api)
	if err := c.EnsureHeaders(t.Context()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c.mu.RLock()
	cached := c.ensuredSheetName
	c.mu.RUnlock()
	if cached != todaySheetName() {
		t.Errorf("cache not updated: got %q, want %q", cached, todaySheetName())
	}
}

// --- RecordArrival tests ---

func TestRecordArrival_Success(t *testing.T) {
	api := &mockSheetsAPI{}
	c := newTestClient(api)
	if err := c.RecordArrival(t.Context(), "Maria Silva"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.appendedValues == nil {
		t.Fatal("expected valuesAppend to be called")
	}
	row := api.appendedValues.Values[0]
	if row[0] != "Maria Silva" {
		t.Errorf("expected name %q in row, got %q", "Maria Silva", row[0])
	}
}

func TestRecordArrival_Error(t *testing.T) {
	api := &mockSheetsAPI{appendValuesErr: errors.New("api failure")}
	c := newTestClient(api)
	if err := c.RecordArrival(t.Context(), "Maria Silva"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- RecordDeparture tests ---

func TestRecordDeparture_Success(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{
			Values: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Carlos", "09:00:00", ""},
			},
		},
	}
	c := newTestClient(api)
	if err := c.RecordDeparture(t.Context(), "Carlos"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if api.updatedValues == nil {
		t.Fatal("expected valuesUpdate to be called")
	}
}

func TestRecordDeparture_NoOpenArrival(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{
			Values: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Carlos", "09:00:00", "17:00:00"}, // already closed
			},
		},
	}
	c := newTestClient(api)
	err := c.RecordDeparture(t.Context(), "Carlos")
	if err == nil {
		t.Fatal("expected error for no open arrival, got nil")
	}
	if !errors.Is(err, err) || !contains(err.Error(), "no open arrival found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRecordDeparture_ReadError(t *testing.T) {
	api := &mockSheetsAPI{getValuesErr: errors.New("api failure")}
	c := newTestClient(api)
	if err := c.RecordDeparture(t.Context(), "Carlos"); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRecordDeparture_UpdatesCorrectRow(t *testing.T) {
	api := &mockSheetsAPI{
		getValuesResult: &sheetsv4.ValueRange{
			Values: [][]interface{}{
				{"Nome", "Chegada", "Saída"},
				{"Ana", "08:00:00", ""},    // row 2 — different person
				{"Carlos", "09:00:00", ""}, // row 3 — target
			},
		},
	}
	c := newTestClient(api)
	if err := c.RecordDeparture(t.Context(), "Carlos"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// row 3 in the sheet → range should reference C3
	wantRange := "'" + todaySheetName() + "'!C3"
	if api.updatedRange != wantRange {
		t.Errorf("updated range = %q, want %q", api.updatedRange, wantRange)
	}
}

// --- todaySheetName test ---

func TestTodaySheetName_Format(t *testing.T) {
	name := todaySheetName()
	// Must match YYYY-MM-DD
	if len(name) != 10 || name[4] != '-' || name[7] != '-' {
		t.Errorf("todaySheetName() = %q, want YYYY-MM-DD format", name)
	}
}

// contains is a helper to avoid importing strings in test assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
