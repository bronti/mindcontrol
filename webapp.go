package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Keep the "filled dates" list in a form URL bounded so it can't grow forever.
const maxFilledDates = 120

// cacheVersion is added to every form URL as ?v=… to bust Telegram's Mini App
// cache. It's the current time, so each open fetches the page and its assets
// fresh — a deployed form change shows up without restarting the bot.
func cacheVersion() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// buildFormURL points at the form in one mode (?form=sleep|day), optionally with a
// pre-selected date and the dates whose matching part is already filled (so the
// form can grey them out).
func buildFormURL(baseURL, part, targetDate string, filled []string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	q.Set("v", cacheVersion())
	q.Set("form", part)
	if targetDate != "" {
		q.Set("date", targetDate)
	}
	if len(filled) > 0 {
		q.Set("filled", strings.Join(recentDates(filled), ","))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// buildEditURL opens the form for one day+part to edit: the date is locked, the
// mode is update|create, and the part's existing values ride along as p_* params.
func buildEditURL(baseURL, part, date string, row []interface{}) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	q.Set("v", cacheVersion())
	q.Set("form", part)
	q.Set("date", date)
	if row != nil && partFilled(row, part) {
		q.Set("mode", "update")
	} else {
		q.Set("mode", "create")
	}

	prefill := func(param, header string) {
		idx := columnIndex(header)
		if row != nil && idx >= 0 && idx < len(row) {
			if v := fmt.Sprint(row[idx]); v != "" {
				q.Set(param, v)
			}
		}
	}
	if part == ownerSleep {
		prefill("p_bedtime", "Fell asleep")
		prefill("p_wake", "Woke up")
		prefill("p_rested", "How rested")
		prefill("p_dreams", "Dreams")
		prefill("p_dream_note", "Dream notes")
		prefill("p_sleep_meds", "Sleep medications")
	} else {
		prefill("p_state", "Overall state")
		prefill("p_anxiety", "Anxiety")
		prefill("p_irritability", "Irritability")
		prefill("p_libido", "Libido")
		prefill("p_drowsiness", "Drowsiness")
		prefill("p_appetite", "Appetite")
		prefill("p_energy", "Energy")
		prefill("p_ate_well", "Ate well")
		prefill("p_menstruation", "Menstruation")
		prefill("p_sex", "Sex")
		prefill("p_masturbation", "Masturbation")
		prefill("p_headache", "Headache")
		prefill("p_smoking", "Smoking")
		prefill("p_meds", "Medications")
		prefill("p_note", "Diary")
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// calendarURL points at the calendar page, carrying the compact per-day data.
func calendarURL(baseURL, days string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.Path += "calendar.html"
	q := u.Query()
	q.Set("v", cacheVersion())
	if days != "" {
		q.Set("days", days)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// recentDates returns the most recent dates (ISO strings sort chronologically).
func recentDates(dates []string) []string {
	sorted := append([]string(nil), dates...)
	sort.Strings(sorted)
	if len(sorted) > maxFilledDates {
		sorted = sorted[len(sorted)-maxFilledDates:]
	}
	return sorted
}
