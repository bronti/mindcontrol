package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"cloud.google.com/go/auth/credentials"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// The Sheets client is created once and reused: building it sets up authentication
// and an HTTP client, which we don't want to redo on every call.
var (
	sheetsOnce sync.Once
	sheetsSvc  *sheets.Service
	sheetsErr  error
)

// service authenticates with Application Default Credentials. DetectDefault looks
// for a key file named by the GOOGLE_APPLICATION_CREDENTIALS environment variable
// first, then falls back to a service account attached to the host (e.g. a GCE VM).
//
// We rely on the KEY FILE in BOTH places we run — set GOOGLE_APPLICATION_CREDENTIALS
// in .env locally AND on the Cloud VM (godotenv loads it before we get here).
//
// Why not the VM's attached service account? Sheets is a Google *Workspace* API,
// and its tokens must carry the `spreadsheets` (or `drive`) scope. A GCE VM's
// attached-SA token uses the instance's access scopes, and the "full access to all
// Cloud APIs" preset grants `cloud-platform` — which covers Google *Cloud* APIs but
// NOT Workspace APIs like Sheets. So the attached-SA path returns
// ACCESS_TOKEN_SCOPE_INSUFFICIENT no matter the scope preset. A key file mints a
// token with exactly the scope requested below, which Sheets accepts.
func service() (*sheets.Service, error) {
	sheetsOnce.Do(func() {
		creds, err := credentials.DetectDefault(&credentials.DetectOptions{
			Scopes: []string{sheets.SpreadsheetsScope},
		})
		if err != nil {
			sheetsErr = err
			return
		}
		sheetsSvc, sheetsErr = sheets.NewService(context.Background(),
			option.WithAuthCredentials(creds))
	})
	if sheetsErr != nil {
		return nil, fmt.Errorf("connecting to Google Sheets: %w", sheetsErr)
	}
	return sheetsSvc, nil
}

// tabRange qualifies an A1 range with the tab name, e.g. tabRange("A2:X") ->
// "'Makhi-Bot'!A2:X". The tab name (sheetTab, in main.go) is single-quoted
// because it contains a hyphen. Change the tab name there, not here.
func tabRange(a1 string) string {
	return fmt.Sprintf("'%s'!%s", sheetTab, a1)
}

// appendRow adds a single row to the end of the tab, leaving existing data alone.
func appendRow(values ...any) error {
	srv, err := service()
	if err != nil {
		return err
	}
	row := &sheets.ValueRange{Values: [][]any{values}}

	// RAW (not USER_ENTERED): store values exactly as given, so date strings stay
	// text that round-trips unchanged and don't get reformatted by the sheet locale.
	_, err = srv.Spreadsheets.Values.
		Append(spreadsheetID, tabRange("A1"), row).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("writing to the sheet: %w", err)
	}
	return nil
}

// lastColumnLetter is the rightmost column of the schema. It assumes ≤ 26 columns
// (A–Z); we have 24, so add proper multi-letter handling only if we pass Z.
func lastColumnLetter() string {
	return string(rune('A' + len(columns) - 1))
}

func dataRange() string {
	return tabRange("A2:" + lastColumnLetter())
}

// readDataRows reads every data row (below the header), with numbers returned as
// numbers (UNFORMATTED) so they round-trip when a row is written back.
func readDataRows() ([][]any, error) {
	srv, err := service()
	if err != nil {
		return nil, err
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

// updateRow overwrites an existing row (1-based) with new values.
func updateRow(rowNumber int, values []any) error {
	srv, err := service()
	if err != nil {
		return err
	}
	rng := tabRange(fmt.Sprintf("A%d:%s%d", rowNumber, lastColumnLetter(), rowNumber))
	_, err = srv.Spreadsheets.Values.
		Update(spreadsheetID, rng, &sheets.ValueRange{Values: [][]any{values}}).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("updating row: %w", err)
	}
	return nil
}

// headerRange is row 1 across every schema column, e.g. "'Makhi-Bot'!A1:X1".
func headerRange() string {
	return tabRange(fmt.Sprintf("A1:%s1", lastColumnLetter()))
}

// readHeaderRow reads the sheet's current header (row 1) so startup can compare it
// against the schema before overwriting. It returns a nil row when the tab has no
// header yet (a fresh sheet); cells come back as strings, ready for headerEqual.
func readHeaderRow() ([]any, error) {
	srv, err := service()
	if err != nil {
		return nil, err
	}
	resp, err := srv.Spreadsheets.Values.
		Get(spreadsheetID, headerRange()).
		Do()
	if err != nil {
		return nil, fmt.Errorf("reading the header row: %w", err)
	}
	if len(resp.Values) == 0 {
		return nil, nil
	}
	return resp.Values[0], nil
}

// syncHeader writes the current column headers into row 1. This is
// non-destructive — row 1 holds only labels, never data — so it keeps the
// sheet's header in step with the code as columns are renamed or appended,
// WITHOUT ever clearing the tab (which now holds real data).
func syncHeader(header []any) error {
	srv, err := service()
	if err != nil {
		return err
	}
	_, err = srv.Spreadsheets.Values.
		Update(spreadsheetID, headerRange(), &sheets.ValueRange{Values: [][]any{header}}).
		ValueInputOption("RAW").
		Do()
	if err != nil {
		return fmt.Errorf("writing the header row: %w", err)
	}
	return nil
}

// --- row-set queries (pure: operate on already-read rows, so one read serves a
// whole keyboard build, and they're unit-testable without the live sheet) ---

// findDateRow returns a date's 1-based sheet row number and its values, or row
// number 0 (and a nil row) if the date isn't among the given rows yet.
func findDateRow(rows [][]any, date string) (int, []any) {
	for i, row := range rows {
		if cellString(row, 0) == date {
			return i + 2, row // +2: data starts at sheet row 2
		}
	}
	return 0, nil
}

// filledByPartRows returns the dates whose sleep part / day part are filled.
func filledByPartRows(rows [][]any) (sleepDates, dayDates []string) {
	for _, row := range rows {
		date := cellString(row, 0)
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
	return sleepDates, dayDates
}

// calendarDataRows encodes the filled days for the calendar view as a compact,
// URL-safe string: entries joined by "_", fields by ".", the date as YYYYMMDD,
// and each part's token = its rating, "f" (part filled but unrated), or "-" (part
// not filled). Only days with something filled are included; capped to the most
// recent days to keep the URL short.
func calendarDataRows(rows [][]any) string {
	restedIdx := columnIndex("How rested")
	stateIdx := columnIndex("Overall state")

	entries := make([]string, 0, len(rows))
	for _, row := range rows {
		date := cellString(row, 0)
		if date == "" {
			continue
		}
		sleepOn := partFilled(row, ownerSleep)
		dayOn := partFilled(row, ownerDay)
		if !sleepOn && !dayOn {
			continue
		}
		top := calToken(sleepOn, cellString(row, restedIdx))
		bottom := calToken(dayOn, cellString(row, stateIdx))
		entries = append(entries, strings.ReplaceAll(date, "-", "")+"."+top+"."+bottom)
	}

	sort.Strings(entries) // YYYYMMDD prefix sorts chronologically
	if len(entries) > maxURLDays {
		entries = entries[len(entries)-maxURLDays:]
	}
	return strings.Join(entries, "_")
}

// latestMedicationsRows returns the medications cell from the most recent entry
// before `before` (an ISO date) that has medications for the given part — so a
// new form can pre-fill the usual drugs at their last-used doses instead of a
// fixed list. The cell is already in the "Name 200mg; Other 3mg" format the form
// parses. Empty string when there's no such entry.
func latestMedicationsRows(rows [][]any, part, before string) string {
	medIdx := columnIndex(medHeader(part))
	bestDate, bestMeds := "", ""
	for _, row := range rows {
		date := cellString(row, 0)
		meds := cellString(row, medIdx)
		if date == "" || meds == "" {
			continue
		}
		if before != "" && date >= before {
			continue // only entries strictly before the day being filled
		}
		if date > bestDate { // ISO dates sort chronologically
			bestDate, bestMeds = date, meds
		}
	}
	return bestMeds
}

// calToken is a part's calendar token: "-" if the part isn't filled, "f" if it's
// filled but has no rating, otherwise the rating value.
func calToken(filled bool, value string) string {
	if !filled {
		return "-"
	}
	if value == "" {
		return "f"
	}
	return value
}

// cellString safely reads a cell as a string ("" if out of range).
func cellString(row []any, idx int) string {
	if idx >= 0 && idx < len(row) {
		return fmt.Sprint(row[idx])
	}
	return ""
}

// medHeader is the sheet column that holds a part's medications.
func medHeader(part string) string {
	if part == ownerSleep {
		return "Sleep medications"
	}
	return "Medications"
}
