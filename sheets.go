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

// dataRange is the full data area of the tab (from row 2 down), across all
// columns. lastColumnLetter assumes ≤ 26 columns (A–Z); we have 23.
func lastColumnLetter() string {
	return string(rune('A' + len(columns) - 1))
}

func dataRange() string {
	return "'Makhi-Bot'!A2:" + lastColumnLetter()
}

// readDataRows reads every data row (below the header), with numbers returned as
// numbers (UNFORMATTED) so they round-trip when a row is written back.
func readDataRows() ([][]interface{}, error) {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google-cloud-key.json"))
	if err != nil {
		return nil, fmt.Errorf("connecting to Google Sheets: %w", err)
	}
	resp, err := srv.Spreadsheets.Values.
		Get(spreadsheetID, dataRange()).
		ValueRenderOption("UNFORMATTED_VALUE").
		Do()
	if err != nil {
		return nil, fmt.Errorf("reading rows: %w", err)
	}
	return resp.Values, nil
}

// findDateRow returns the 1-based sheet row number and current values for a date,
// or row number 0 if the date isn't in the tab yet.
func findDateRow(date string) (int, []interface{}, error) {
	rows, err := readDataRows()
	if err != nil {
		return 0, nil, err
	}
	for i, row := range rows {
		if len(row) > 0 && fmt.Sprint(row[0]) == date {
			return i + 2, row, nil // +2: data starts at sheet row 2
		}
	}
	return 0, nil, nil
}

// updateRow overwrites an existing row (1-based) with new values.
func updateRow(rowNumber int, values []interface{}) error {
	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google-cloud-key.json"))
	if err != nil {
		return fmt.Errorf("connecting to Google Sheets: %w", err)
	}
	rng := fmt.Sprintf("'Makhi-Bot'!A%d:%s%d", rowNumber, lastColumnLetter(), rowNumber)
	_, err = srv.Spreadsheets.Values.
		Update(spreadsheetID, rng, &sheets.ValueRange{Values: [][]interface{}{values}}).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("updating row: %w", err)
	}
	return nil
}

// filledByPart reads all rows once and returns the dates where the sleep part and
// the day part are already filled. Used to grey out days in each form.
func filledByPart() (sleepDates, dayDates []string, err error) {
	rows, err := readDataRows()
	if err != nil {
		return nil, nil, err
	}
	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		date := fmt.Sprint(row[0])
		if date == "" {
			continue
		}
		if partFilled(row, ownerSleep) {
			sleepDates = append(sleepDates, date)
		}
		if partFilled(row, ownerDay) {
			dayDates = append(dayDates, date)
		}
	}
	return sleepDates, dayDates, nil
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
