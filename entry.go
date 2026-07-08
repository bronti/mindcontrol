package main

import (
	"fmt"
	"strings"
)

type medication struct {
	Name string `json:"name"`
	Dose string `json:"dose"` // milligrams; may be empty or a decimal like "12.5"
}

// formAnswers mirrors the JSON the Mini App sends via tg.sendData (see docs/app.js);
// the json tags must match the keys there. Pointer fields stay nil when a value was
// never entered, so "blank" stays distinct from a real 0.
type formAnswers struct {
	FormType         string       `json:"form_type"` // "sleep" or "day"
	Edit             bool         `json:"edit"`      // editing an existing entry (overwrite allowed)
	Date             string       `json:"date"`
	LastModified     string       `json:"-"` // set by the bot at submit time (also on edits)
	Bedtime          string       `json:"bedtime"`
	Wake             string       `json:"wake"`
	SleepHours       *float64     `json:"sleep_hours"`
	Rested           *int         `json:"rested"`
	Dreams           string       `json:"dreams"`
	DreamNote        string       `json:"dream_note"`
	SleepMedications []medication `json:"sleep_medications"`
	State            *int         `json:"state"`
	Anxiety          *int         `json:"anxiety"`
	Irritability     *int         `json:"irritability"`
	Libido           *int         `json:"libido"`
	Drowsiness       *int         `json:"drowsiness"`
	Appetite         *int         `json:"appetite"`
	Energy           *int         `json:"energy"`
	AteWell          *int         `json:"ate_well"`
	Menstruation     bool         `json:"menstruation"`
	Sex              bool         `json:"sex"`
	Masturbation     bool         `json:"masturbation"`
	Headache         bool         `json:"headache"`
	Smoking          bool         `json:"smoking"`
	Medications      []medication `json:"medications"`
	Note             string       `json:"note"`
}

// Each column is owned by the form that fills it. Meta columns are written on every
// submission; sleep/day columns only by their own form (see mergeRow).
const (
	ownerMeta  = "meta"
	ownerSleep = "sleep"
	ownerDay   = "day"
)

// columns is the single source of truth for the Makhi-Bot tab: this order defines
// both the header row and every data row, grouped date | sleep… | day… | filled-at.
// Only reorder while the tab is empty — a reorder misaligns existing rows.
var columns = []struct {
	header string
	owner  string
	value  func(a formAnswers) interface{}
}{
	{"Date", ownerMeta, func(a formAnswers) interface{} { return a.Date }},
	{"Fell asleep", ownerSleep, func(a formAnswers) interface{} { return a.Bedtime }},
	{"Woke up", ownerSleep, func(a formAnswers) interface{} { return a.Wake }},
	{"Sleep hours", ownerSleep, func(a formAnswers) interface{} { return sleepCell(a.SleepHours) }},
	{"How rested", ownerSleep, func(a formAnswers) interface{} { return numCell(a.Rested) }},
	{"Dreams", ownerSleep, func(a formAnswers) interface{} { return a.Dreams }},
	{"Dream notes", ownerSleep, func(a formAnswers) interface{} { return dreamNote(a) }},
	{"Sleep medications", ownerSleep, func(a formAnswers) interface{} { return formatMedications(a.SleepMedications) }},
	{"Overall state", ownerDay, func(a formAnswers) interface{} { return numCell(a.State) }},
	{"Anxiety", ownerDay, func(a formAnswers) interface{} { return numCell(a.Anxiety) }},
	{"Irritability", ownerDay, func(a formAnswers) interface{} { return numCell(a.Irritability) }},
	{"Libido", ownerDay, func(a formAnswers) interface{} { return numCell(a.Libido) }},
	{"Drowsiness", ownerDay, func(a formAnswers) interface{} { return numCell(a.Drowsiness) }},
	{"Appetite", ownerDay, func(a formAnswers) interface{} { return numCell(a.Appetite) }},
	{"Energy", ownerDay, func(a formAnswers) interface{} { return numCell(a.Energy) }},
	{"Ate well", ownerDay, func(a formAnswers) interface{} { return numCell(a.AteWell) }},
	{"Menstruation", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Menstruation) }},
	{"Sex", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Sex) }},
	{"Masturbation", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Masturbation) }},
	{"Headache", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Headache) }},
	{"Smoking", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Smoking) }},
	{"Medications", ownerDay, func(a formAnswers) interface{} { return formatMedications(a.Medications) }},
	{"Diary", ownerDay, func(a formAnswers) interface{} { return a.Note }},
	{"Last modified", ownerMeta, func(a formAnswers) interface{} { return a.LastModified }},
}

func headerRow() []interface{} {
	row := make([]interface{}, len(columns))
	for i, c := range columns {
		row[i] = c.header
	}
	return row
}

// mergeRow overlays one form's answers onto an existing row (nil for a new day):
// columns owned by the submitting part — plus meta columns — get fresh values;
// every other column keeps what was already there. This is what lets the sleep and
// day forms fill the same row at different times.
func mergeRow(existing []interface{}, a formAnswers, part string) []interface{} {
	row := make([]interface{}, len(columns))
	for i := range columns {
		if i < len(existing) {
			row[i] = existing[i]
		} else {
			row[i] = ""
		}
	}
	for i, c := range columns {
		if c.owner == part || c.owner == ownerMeta {
			row[i] = c.value(a)
		}
	}
	return row
}

func columnIndex(header string) int {
	for i, c := range columns {
		if c.header == header {
			return i
		}
	}
	return -1
}

func partFilled(row []interface{}, part string) bool {
	for i, c := range columns {
		if c.owner == part && i < len(row) && fmt.Sprint(row[i]) != "" {
			return true
		}
	}
	return false
}

// --- how each field renders into a sheet cell ---

// dreamNote drops the text unless there actually were dreams, so text typed and
// then dismissed (dreams set back to "none") is never saved.
func dreamNote(a formAnswers) string {
	switch a.Dreams {
	case "dreams", "nightmares", "anxious":
		return a.DreamNote
	}
	return ""
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// sleepCell / numCell render a pointer value, or "" when it was never set (nil) —
// so an untouched slider or a missing sleep time stays blank rather than 0.
func sleepCell(hours *float64) interface{} {
	if hours == nil {
		return ""
	}
	return *hours
}

func numCell(n *int) interface{} {
	if n == nil {
		return ""
	}
	return *n
}

func formatMedications(meds []medication) string {
	parts := make([]string, 0, len(meds))
	for _, m := range meds {
		if m.Dose != "" {
			parts = append(parts, fmt.Sprintf("%s %smg", m.Name, m.Dose))
		} else {
			parts = append(parts, m.Name)
		}
	}
	return strings.Join(parts, "; ")
}
