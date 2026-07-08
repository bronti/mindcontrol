package main

import "testing"

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
	if len(columns) != 23 {
		t.Fatalf("expected 23 columns, got %d", len(columns))
	}
}

// The core of the split: a Sleep submission fills only sleep columns, and a later
// Day submission merges into the same row without touching the sleep columns.
func TestMergeSleepThenDay(t *testing.T) {
	sleep := formAnswers{
		Date:         "2026-07-08",
		FilledAt:     "2026-07-08 08:00:00",
		Bedtime:      "23:30",
		Wake:         "07:00",
		SleepQuality: 3,
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
		Date:     "2026-07-08",
		FilledAt: "2026-07-08 21:30:00",
		State:    7,
		Headache: true,
		Smoking:  true,
		Note:     "long day",
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
	if got := row2[colIndex("Filled at")]; got != "2026-07-08 21:30:00" {
		t.Errorf("filled-at not updated: %v", got)
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

// Missing sleep times must leave the duration cell empty, not 0.
func TestSleepCellEmptyWhenNoTimes(t *testing.T) {
	var a formAnswers
	if got := sleepCell(a.SleepHours); got != "" {
		t.Errorf("expected empty sleep cell, got %v", got)
	}
}
