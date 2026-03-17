package sheets

import (
	"context"

	sheetsv4 "google.golang.org/api/sheets/v4"
)

type mockSheetsAPI struct {
	batchUpdateErr  error
	batchUpdateFunc func(sheetID string, req *sheetsv4.BatchUpdateSpreadsheetRequest) error

	valuesGetResult *sheetsv4.ValueRange
	valuesGetErr    error

	valuesUpdateErr  error
	valuesUpdateFunc func(sheetID, cellRange string, values *sheetsv4.ValueRange) error

	valuesAppendErr  error
	valuesAppendFunc func(sheetID, cellRange string, values *sheetsv4.ValueRange) error

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

func (m *mockSheetsAPI) getValues(ctx context.Context, sheetID, cellRange string) (*sheetsv4.ValueRange, error) {
	if m.valuesGetErr != nil {
		return nil, m.valuesGetErr
	}
	if m.valuesGetResult != nil {
		return m.valuesGetResult, nil
	}
	return &sheetsv4.ValueRange{}, nil
}

func (m *mockSheetsAPI) updateValues(ctx context.Context, sheetID, cellRange string, values *sheetsv4.ValueRange) error {
	m.updatedRange = cellRange
	m.updatedValues = values
	if m.valuesUpdateFunc != nil {
		return m.valuesUpdateFunc(sheetID, cellRange, values)
	}
	return m.valuesUpdateErr
}

func (m *mockSheetsAPI) appendValues(ctx context.Context, sheetID, cellRange string, values *sheetsv4.ValueRange) error {

}
