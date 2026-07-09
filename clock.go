package main

import (
	"log"
	"time"
	// Embed the IANA time zone database in the binary so LoadLocation always works,
	// even on a host that has no system tzdata installed (a minimal VM or container).
	_ "time/tzdata"
)

// loadLocation resolves an IANA time zone name (e.g. "Europe/Moscow", set via
// TIMEZONE in .env) to a *time.Location. It's used for every wall-clock decision:
// when reminders fire and what counts as "today"/"yesterday". An empty or invalid
// value falls back to the machine's own local time — the bot still runs, but note
// a cloud VM's clock is usually UTC, which is exactly why setting TIMEZONE matters
// once the bot is hosted.
func loadLocation(name string) *time.Location {
	if name == "" {
		log.Print("TIMEZONE not set — using this machine's local time for reminders and dates")
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Printf("TIMEZONE %q is not a valid IANA name (e.g. Europe/Moscow) — using local time instead: %v", name, err)
		return time.Local
	}
	log.Printf("using time zone %s for reminders and dates", name)
	return loc
}

// now is the current time in the configured time zone. Everything date- or
// time-of-day related goes through it, so the bot behaves the same whatever the
// host clock is set to.
func (s *server) now() time.Time {
	return time.Now().In(s.location)
}
