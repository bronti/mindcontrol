package main

import (
	"strings"
	"testing"
)


// buildEditURL must carry the mode, the part's values as p_* params, the date,
// and the drug catalog (?meds=) so the picker works while editing.
func TestBuildEditURL(t *testing.T) {
	row := mergeRow(nil, formAnswers{Date: "2026-07-03", State: new(7), Menstruation: true}, ownerDay)
	u := buildEditURL("https://x/", ownerDay, "2026-07-03", row, "", "Aspirin 100mg")
	for _, want := range []string{"form=day", "mode=update", "date=2026-07-03", "p_state=7", "p_menstruation=yes", "meds=Aspirin"} {
		if !strings.Contains(u, want) {
			t.Errorf("edit URL %q missing %q", u, want)
		}
	}
	// A day with no entry yet → create mode, no pre-fill params, but the usual
	// meds still ride along as def_meds.
	empty := buildEditURL("https://x/", ownerSleep, "2026-07-04", nil, "Melatonin 3mg", "")
	if !strings.Contains(empty, "mode=create") || strings.Contains(empty, "p_") {
		t.Errorf("expected create mode with no p_ params, got %q", empty)
	}
	if !strings.Contains(empty, "def_meds=Melatonin") {
		t.Errorf("expected def_meds pre-fill in create mode, got %q", empty)
	}
	if strings.Contains(empty, "&meds=") {
		t.Errorf("expected no meds param when the catalog is empty, got %q", empty)
	}
}

// A normal form open carries the drug catalog as ?meds= and the most-recent
// medications as def_meds; both params are omitted entirely when empty.
func TestBuildFormURLMedications(t *testing.T) {
	u := buildFormURL("https://x/", ownerDay, "", nil, "Melatonin 3mg; Aspirin 100mg", "Aspirin 100mg; Paracetamol")
	for _, want := range []string{"form=day", "def_meds=Melatonin", "meds=Aspirin"} {
		if !strings.Contains(u, want) {
			t.Errorf("form URL %q missing %q", u, want)
		}
	}
	got := buildFormURL("https://x/", ownerSleep, "", nil, "", "")
	if strings.Contains(got, "meds") { // also catches def_meds
		t.Errorf("expected no medication params when empty, got %q", got)
	}
}

// The row-set helpers that back the keyboard now run on plain data, so we can
// test their logic without touching the live sheet.

func TestFilledByPartRows(t *testing.T) {
	rows := [][]any{
		mergeRow(nil, formAnswers{Date: "2026-07-01", Bedtime: "23:00"}, ownerSleep), // sleep only
		mergeRow(nil, formAnswers{Date: "2026-07-02", State: new(5)}, ownerDay),      // day only
	}
	sleep, day := filledByPartRows(rows)
	if len(sleep) != 1 || sleep[0] != "2026-07-01" {
		t.Errorf("sleep dates: got %v, want [2026-07-01]", sleep)
	}
	if len(day) != 1 || day[0] != "2026-07-02" {
		t.Errorf("day dates: got %v, want [2026-07-02]", day)
	}
}

func TestCalendarDataRows(t *testing.T) {
	rows := [][]any{
		mergeRow(nil, formAnswers{Date: "2026-07-02", State: new(7)}, ownerDay),
		mergeRow(nil, formAnswers{Date: "2026-07-01", Rested: new(3)}, ownerSleep),
	}
	// Sorted chronologically; sleep=top (rating), day=bottom, "-" for the missing half.
	if got, want := calendarDataRows(rows), "20260701.3.-_20260702.-.7"; got != want {
		t.Errorf("calendar data: got %q, want %q", got, want)
	}
}

func TestLatestMedicationsRows(t *testing.T) {
	med := func(name, dose string) []medication { return []medication{{Name: name, Dose: dose}} }
	rows := [][]any{
		mergeRow(nil, formAnswers{Date: "2026-07-01", Medications: med("Aspirin", "100")}, ownerDay),
		mergeRow(nil, formAnswers{Date: "2026-07-05", Medications: med("Melatonin", "5")}, ownerDay),
		mergeRow(nil, formAnswers{Date: "2026-07-03", Medications: med("Paracetamol", "500")}, ownerDay),
	}
	if got := latestMedicationsRows(rows, ownerDay, "2026-07-10"); got != "Melatonin 5mg" {
		t.Errorf("latest before 07-10: got %q, want Melatonin 5mg", got)
	}
	// 07-05 is excluded (>= before), so the newest remaining is 07-03.
	if got := latestMedicationsRows(rows, ownerDay, "2026-07-04"); got != "Paracetamol 500mg" {
		t.Errorf("latest before 07-04: got %q, want Paracetamol 500mg", got)
	}
	// These rows have no sleep medications.
	if got := latestMedicationsRows(rows, ownerSleep, "2026-07-10"); got != "" {
		t.Errorf("latest sleep meds: got %q, want empty", got)
	}
}

func TestHeaderRowMatchesColumns(t *testing.T) {
	if len(headerRow()) != len(columns) {
		t.Fatalf("header has %d entries but columns has %d", len(headerRow()), len(columns))
	}
	if len(columns) != 24 {
		t.Fatalf("expected 24 columns, got %d", len(columns))
	}
}

// The startup header guard rests on these two pure checks: an empty existing
// header is safe to write, and only a same-length, cell-for-cell match counts as
// equal (so a rename or an added/removed column trips the pause).
func TestHeaderEmptyAndEqual(t *testing.T) {
	if !headerEmpty(nil) {
		t.Error("nil header should be empty")
	}
	if !headerEmpty([]any{"", "  ", ""}) {
		t.Error("all-blank header should be empty")
	}
	if headerEmpty([]any{"", "Date"}) {
		t.Error("header with a label should not be empty")
	}

	want := headerRow()
	if !headerEqual(want, want) {
		t.Error("a header should equal itself")
	}
	// A fresh read comes back as strings; the schema stores strings too, but
	// compare a rebuilt string slice to be explicit about the string path.
	sameStrings := make([]any, len(want))
	for i, c := range want {
		sameStrings[i] = c.(string)
	}
	if !headerEqual(want, sameStrings) {
		t.Error("equal cell values should compare equal")
	}
	if headerEqual(want, want[:len(want)-1]) {
		t.Error("a shorter header should not be equal (a column was dropped)")
	}
	renamed := append([]any(nil), want...)
	renamed[0] = "Datum"
	if headerEqual(want, renamed) {
		t.Error("a renamed column should not be equal")
	}
}

// The core of the split: a Sleep submission fills only sleep columns, and a later
// Day submission merges into the same row without touching the sleep columns.
func TestMergeSleepThenDay(t *testing.T) {
	sleep := formAnswers{
		Date:         "2026-07-08",
		LastModified: "2026-07-08 08:00:00",
		Bedtime:      "23:30",
		Wake:         "07:00",
		Rested:       new(3),
		Dreams:       "nightmares",
		DreamNote:    "chased by a dog",
	}
	row := mergeRow(nil, sleep, ownerSleep)

	if got := row[columnIndex("Fell asleep")]; got != "23:30" {
		t.Errorf("bedtime not written: %v", got)
	}
	if got := row[columnIndex("Dream notes")]; got != "chased by a dog" {
		t.Errorf("dream note not written: %v", got)
	}
	if got := row[columnIndex("Overall state")]; got != "" {
		t.Errorf("day column should stay empty after a sleep submit, got %v", got)
	}
	if !partFilled(row, ownerSleep) {
		t.Error("sleep part should be filled")
	}
	if partFilled(row, ownerDay) {
		t.Error("day part should not be filled yet")
	}

	day := formAnswers{
		Date:         "2026-07-08",
		LastModified: "2026-07-08 21:30:00",
		State:        new(7),
		Headache:     true,
		Smoking:      true,
		Note:         "long day",
	}
	row2 := mergeRow(row, day, ownerDay)

	if got := row2[columnIndex("Fell asleep")]; got != "23:30" {
		t.Errorf("sleep column lost after day merge: %v", got)
	}
	if got := row2[columnIndex("Overall state")]; got != 7 {
		t.Errorf("state not written: %v", got)
	}
	if got := row2[columnIndex("Smoking")]; got != "yes" {
		t.Errorf("smoking not written: %v", got)
	}
	if got := row2[columnIndex("Last modified")]; got != "2026-07-08 21:30:00" {
		t.Errorf("last-modified not updated: %v", got)
	}
	if !partFilled(row2, ownerSleep) || !partFilled(row2, ownerDay) {
		t.Error("both parts should be filled after both submits")
	}
}

// Dream text typed and then dismissed (dreams switched back to "none") must not
// be saved.
func TestDreamNoteDroppedWhenNone(t *testing.T) {
	a := formAnswers{Dreams: "none", DreamNote: "typed then changed my mind"}
	if got := dreamNote(a); got != "" {
		t.Errorf("expected dream note dropped for none, got %q", got)
	}
	a.Dreams = "dreams"
	if got := dreamNote(a); got != "typed then changed my mind" {
		t.Errorf("expected dream note kept for dreams, got %q", got)
	}
}

// Calendar tokens: "-" when the part isn't filled, "f" when filled but unrated,
// otherwise the rating value.
func TestCalToken(t *testing.T) {
	if got := calToken(false, "3"); got != "-" {
		t.Errorf("not filled: got %q, want -", got)
	}
	if got := calToken(true, ""); got != "f" {
		t.Errorf("filled unrated: got %q, want f", got)
	}
	if got := calToken(true, "4"); got != "4" {
		t.Errorf("filled rated: got %q, want 4", got)
	}
}

// An untouched slider (nil) must be blank, but a touched 0 is a real answer.
func TestNumCellEmptyWhenNil(t *testing.T) {
	if got := numCell(nil); got != "" {
		t.Errorf("expected empty for an untouched slider, got %v", got)
	}
	if got := numCell(new(0)); got != 0 {
		t.Errorf("expected 0 for a touched-to-zero slider, got %v", got)
	}
}

// Missing sleep times must leave the duration cell empty, not 0.
func TestSleepCellEmptyWhenNoTimes(t *testing.T) {
	var a formAnswers
	if got := sleepCell(a.SleepHours); got != "" {
		t.Errorf("expected empty sleep cell, got %v", got)
	}
}
