package main

import (
	"encoding/json"
	"testing"
)

// A payload shaped exactly like what docs/app.js sends via tg.sendData.
const sampleForm = `{
  "date": "2026-07-07",
  "bedtime": "23:30",
  "wake": "07:00",
  "sleep_hours": 7.5,
  "sleep_quality": 8,
  "dreams": "nightmares",
  "dream_note": "chased by a dog",
  "state": 7,
  "anxiety": 3,
  "irritability": 2,
  "libido": 5,
  "drowsiness": 4,
  "appetite": 6,
  "energy": 7,
  "ate_well": 9,
  "menstruation": false,
  "sex": true,
  "masturbation": false,
  "headache": true,
  "smoking": true,
  "medications": [
    {"name": "Lamotrigine", "dose": "100"},
    {"name": "Fluoxetine", "dose": "20"}
  ],
  "note": "long day but fine"
}`

func TestFormAnswersRow(t *testing.T) {
	var a formAnswers
	if err := json.Unmarshal([]byte(sampleForm), &a); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	a.FilledAt = "2026-07-07 09:15:00"
	row := a.row()

	if len(row) != 23 {
		t.Fatalf("expected 23 columns, got %d", len(row))
	}

	// Spot-check the columns that go through a transform.
	checks := map[int]interface{}{
		0:  "2026-07-07",                         // date (from the form)
		3:  7.5,                                  // sleep hours
		4:  8,                                    // sleep quality
		5:  "nightmares",                         // dreams
		15: "yes",                                // sex -> yes
		14: "no",                                 // menstruation -> no
		17: "yes",                                // headache -> yes
		18: "Lamotrigine 100mg; Fluoxetine 20mg", // medications
		19: "long day but fine",                  // note
		20: "2026-07-07 09:15:00",                // filled-at timestamp
		21: "chased by a dog",                    // dream notes (dreams were present)
		22: "yes",                                // smoking -> yes
	}
	for i, want := range checks {
		if row[i] != want {
			t.Errorf("column %d: got %v (%T), want %v (%T)", i, row[i], row[i], want, want)
		}
	}
}

// The header and a data row must always have the same number of columns, so
// values never land under the wrong header.
func TestHeaderAndRowAligned(t *testing.T) {
	if got, want := len(formAnswers{}.row()), len(headerRow()); got != want {
		t.Fatalf("row has %d columns but header has %d", got, want)
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
	if err := json.Unmarshal([]byte(`{"sleep_hours": null}`), &a); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if got := sleepCell(a.SleepHours); got != "" {
		t.Errorf("expected empty sleep cell, got %v", got)
	}
}
