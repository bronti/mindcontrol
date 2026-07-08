package main

import (
	"strings"
	"testing"
)

func ptr(i int) *int { return &i }

// buildEditURL must carry the mode, the part's values as p_* params, and the date.
func TestBuildEditURL(t *testing.T) {
	row := mergeRow(nil, formAnswers{Date: "2026-07-03", State: ptr(7), Menstruation: true}, ownerDay)
	u := buildEditURL("https://x/", ownerDay, "2026-07-03", row, "")
	for _, want := range []string{"form=day", "mode=update", "date=2026-07-03", "p_state=7", "p_menstruation=yes"} {
		if !strings.Contains(u, want) {
			t.Errorf("edit URL %q missing %q", u, want)
		}
	}
	// A day with no entry yet → create mode, no pre-fill params, but the usual
	// meds still ride along as def_meds.
	empty := buildEditURL("https://x/", ownerSleep, "2026-07-04", nil, "Lamotrigine 200mg")
	if !strings.Contains(empty, "mode=create") || strings.Contains(empty, "p_") {
		t.Errorf("expected create mode with no p_ params, got %q", empty)
	}
	if !strings.Contains(empty, "def_meds=Lamotrigine") {
		t.Errorf("expected def_meds pre-fill in create mode, got %q", empty)
	}
}

// A normal form open carries the most-recent meds as def_meds, and omits the
// param entirely when there are none.
func TestBuildFormURLDefaultMeds(t *testing.T) {
	u := buildFormURL("https://x/", ownerDay, "", nil, "Lamotrigine 200mg; Olanzapine 3mg")
	for _, want := range []string{"form=day", "def_meds=Lamotrigine"} {
		if !strings.Contains(u, want) {
			t.Errorf("form URL %q missing %q", u, want)
		}
	}
	if got := buildFormURL("https://x/", ownerSleep, "", nil, ""); strings.Contains(got, "def_meds") {
		t.Errorf("expected no def_meds when empty, got %q", got)
	}
}

// colIndex finds a column by its header (so tests don't break on a reorder).
func colIndex(header string) int {
	for i, c := range columns {
		if c.header == header {
			return i
		}
	}
	return -1
}

func TestHeaderRowMatchesColumns(t *testing.T) {
	if len(headerRow()) != len(columns) {
		t.Fatalf("header has %d entries but columns has %d", len(headerRow()), len(columns))
	}
	if len(columns) != 24 {
		t.Fatalf("expected 24 columns, got %d", len(columns))
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
		Rested:       ptr(3),
		Dreams:       "nightmares",
		DreamNote:    "chased by a dog",
	}
	row := mergeRow(nil, sleep, ownerSleep)

	if got := row[colIndex("Fell asleep")]; got != "23:30" {
		t.Errorf("bedtime not written: %v", got)
	}
	if got := row[colIndex("Dream notes")]; got != "chased by a dog" {
		t.Errorf("dream note not written: %v", got)
	}
	if got := row[colIndex("Overall state")]; got != "" {
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
		State:        ptr(7),
		Headache:     true,
		Smoking:      true,
		Note:         "long day",
	}
	row2 := mergeRow(row, day, ownerDay)

	if got := row2[colIndex("Fell asleep")]; got != "23:30" {
		t.Errorf("sleep column lost after day merge: %v", got)
	}
	if got := row2[colIndex("Overall state")]; got != 7 {
		t.Errorf("state not written: %v", got)
	}
	if got := row2[colIndex("Smoking")]; got != "yes" {
		t.Errorf("smoking not written: %v", got)
	}
	if got := row2[colIndex("Last modified")]; got != "2026-07-08 21:30:00" {
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
	if got := numCell(ptr(0)); got != 0 {
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
