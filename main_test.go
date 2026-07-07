package main

import (
	"encoding/json"
	"testing"
)

// A payload shaped exactly like what docs/app.js sends via tg.sendData.
const sampleForm = `{
  "bedtime": "23:30",
  "wake": "07:00",
  "sleep_hours": 7.5,
  "sleep_quality": 8,
  "dreams": "nightmares",
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

	row := a.row("2026-07-07")

	if len(row) != 20 {
		t.Fatalf("expected 20 columns, got %d", len(row))
	}

	// Spot-check the columns that go through a transform.
	checks := map[int]interface{}{
		0:  "2026-07-07",                    // date
		3:  7.5,                             // sleep hours
		4:  8,                               // sleep quality
		5:  "nightmares",                    // dreams
		15: "yes",                           // sex -> yes
		14: "no",                            // menstruation -> no
		17: "yes",                           // headache -> yes
		18: "Lamotrigine 100mg; Fluoxetine 20mg", // medications
		19: "long day but fine",            // note
	}
	for i, want := range checks {
		if row[i] != want {
			t.Errorf("column %d: got %v (%T), want %v (%T)", i, row[i], row[i], want, want)
		}
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
