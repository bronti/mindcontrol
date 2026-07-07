package main

import (
	"encoding/json"
	"os"
	"sync"
)

// settings.json holds the bot's local runtime state: which chat to send
// reminders to, and at what times. It's gitignored (the chat id is private).
const settingsFile = "settings.json"

type settings struct {
	ChatID            int64  `json:"chat_id"`            // where to send reminders (learned from messages)
	EveningReminder   string `json:"evening_reminder"`   // "HH:MM", the daily nudge
	AfternoonReminder string `json:"afternoon_reminder"` // "HH:MM", the next-day catch-up
}

var (
	settingsMu    sync.RWMutex
	currentConfig settings
)

// loadSettings reads settings.json into memory, filling in defaults for anything
// missing (including when the file doesn't exist yet).
func loadSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	currentConfig = settings{EveningReminder: "21:00", AfternoonReminder: "14:00"}
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		return // no file yet — defaults stand
	}
	_ = json.Unmarshal(data, &currentConfig)
	if currentConfig.EveningReminder == "" {
		currentConfig.EveningReminder = "21:00"
	}
	if currentConfig.AfternoonReminder == "" {
		currentConfig.AfternoonReminder = "14:00"
	}
}

// getSettings returns a copy of the current settings, safe to read from any goroutine.
func getSettings() settings {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	return currentConfig
}

// saveSettings applies an update under lock and writes the file back to disk.
func saveSettings(update func(*settings)) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	update(&currentConfig)
	data, err := json.MarshalIndent(currentConfig, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0o644)
}
