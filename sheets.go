package main

import (
	"context"
	"fmt"
	"log"

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

	// RAW (not USER_ENTERED): store values exactly as given, so date strings stay
	// text that round-trips unchanged and don't get reformatted by the sheet locale.
	_, err = srv.Spreadsheets.Values.
		Append(spreadsheetID, sheetRange, row).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("writing to the sheet: %w", err)
	}
	return nil
}

// existingDates returns the dates already saved in column A of the Makhi-Bot tab
// (row 1, the header, is skipped). Used to stop the same day being filled twice.
func existingDates() ([]string, error) {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google-cloud-key.json"))
	if err != nil {
		return nil, fmt.Errorf("connecting to Google Sheets: %w", err)
	}

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, "'Makhi-Bot'!A2:A").Do()
	if err != nil {
		return nil, fmt.Errorf("reading existing dates: %w", err)
	}

	dates := make([]string, 0, len(resp.Values))
	for _, row := range resp.Values {
		if len(row) > 0 {
			if s := fmt.Sprint(row[0]); s != "" {
				dates = append(dates, s)
			}
		}
	}
	return dates, nil
}

// ensureHeader writes the header row to the top of the Makhi-Bot tab, but only
// when the tab is empty. If row 1 already has content it is left untouched (so we
// never clobber existing data); a mismatch is logged so you can clear the tab and
// let the bot write a fresh header if you want.
func ensureHeader(header []interface{}) error {
	ctx := context.Background()

	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google-cloud-key.json"))
	if err != nil {
		return fmt.Errorf("connecting to Google Sheets: %w", err)
	}

	resp, err := srv.Spreadsheets.Values.Get(spreadsheetID, "'Makhi-Bot'!1:1").Do()
	if err != nil {
		return fmt.Errorf("reading the header row: %w", err)
	}

	// Row 1 already has something in it — don't overwrite it.
	if len(resp.Values) > 0 && len(resp.Values[0]) > 0 {
		if !headerMatches(resp.Values[0], header) {
			log.Print("warning: the Makhi-Bot tab's first row doesn't match the expected header — " +
				"clear the tab if you want the bot to write a fresh header")
		}
		return nil
	}

	// Empty tab: write the header as row 1.
	_, err = srv.Spreadsheets.Values.
		Update(spreadsheetID, "'Makhi-Bot'!A1", &sheets.ValueRange{Values: [][]interface{}{header}}).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("writing the header row: %w", err)
	}
	log.Print("Wrote the header row to the Makhi-Bot tab")
	return nil
}

// headerMatches reports whether the sheet's first row equals the expected header.
func headerMatches(got, want []interface{}) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range want {
		if fmt.Sprint(got[i]) != fmt.Sprint(want[i]) {
			return false
		}
	}
	return true
}
