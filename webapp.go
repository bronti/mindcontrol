package main

import (
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// maxURLDays caps every per-day list that rides in a Mini App URL (the form's
// filled dates, the calendar's day data) at the most recent N days, so the URLs
// can't grow forever as entries accumulate.
const maxURLDays = 120

// cacheVersion is added to every form URL as ?v=… to bust Telegram's Mini App
// cache. It's the current time, so each open fetches the page and its assets
// fresh — a deployed form change shows up without restarting the bot.
func cacheVersion() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// buildFormURL points at the form in one mode (?form=sleep|day), optionally with a
// pre-selected date, the dates whose matching part is already filled (so the form
// can grey them out), the medications to pre-fill from the most recent entry, and
// the catalog of drugs the picker offers. The catalog rides in the URL (?meds=)
// because it's personal: it lives in the bot's .env, never in the public page.
func buildFormURL(baseURL, part, targetDate string, filled []string, defaultMeds, catalog string) string {
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
	if defaultMeds != "" {
		q.Set("def_meds", defaultMeds)
	}
	if catalog != "" {
		q.Set("meds", catalog)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// buildEditURL opens the form for one day+part to edit: the date is locked, the
// mode is update|create, and the part's existing values ride along as p_* params.
// The drug catalog (?meds=) is always included so the picker works while editing.
func buildEditURL(baseURL, part, date string, row []any, defaultMeds, catalog string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	q.Set("v", cacheVersion())
	q.Set("form", part)
	q.Set("date", date)
	if catalog != "" {
		q.Set("meds", catalog)
	}
	if row != nil && partFilled(row, part) {
		q.Set("mode", "update")
	} else {
		q.Set("mode", "create")
		// A brand-new entry from the calendar still gets the usual meds pre-filled.
		if defaultMeds != "" {
			q.Set("def_meds", defaultMeds)
		}
	}

	// Pre-fill the part's saved values: every column that has an editParam rides
	// along as ?p_*=… (which params exist per column is part of the schema).
	for i, c := range columns {
		if c.owner != part || c.editParam == "" || i >= len(row) {
			continue
		}
		if v := fmt.Sprint(row[i]); v != "" {
			q.Set(c.editParam, v)
		}
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
	if len(sorted) > maxURLDays {
		sorted = sorted[len(sorted)-maxURLDays:]
	}
	return sorted
}
