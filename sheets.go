package main

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// ID of the Google Sheet (from the URL, between /d/ and /edit).
const spreadsheetID = "1bpCNYzsXwgHFLL4ylm3g3Smsb140kMUYKx2zcViEZAw"

// The tab (worksheet) the bot writes to. Quoted because of the hyphen in the name.
const sheetRange = "'Makhi-Bot'!A1"

// appendRow adds a single row to the end of the Makhi-Bot tab.
// Existing data is left untouched — this only appends at the end.
//
// Example: appendRow("2026-07-03", "good", "slept well")
//
// Kept ready for later — we'll wire it up once the bot starts
// collecting answers to the questions.
func appendRow(values ...interface{}) error {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google-cloud-key.json"))
	if err != nil {
		return fmt.Errorf("connecting to Google Sheets: %w", err)
	}

	row := &sheets.ValueRange{Values: [][]interface{}{values}}

	_, err = srv.Spreadsheets.Values.
		Append(spreadsheetID, sheetRange, row).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return fmt.Errorf("writing to the sheet: %w", err)
	}
	return nil
}
