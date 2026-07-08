package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// runReminders checks the local clock every ~20s and fires at most once per day
// per reminder. It uses the machine's local time, so reminders only happen while
// the bot (and this computer) are running.
func (s *server) runReminders() {
	var lastEvening, lastAfternoon string // the date each reminder last fired on

	for {
		now := time.Now()
		hhmm := now.Format("15:04")
		today := now.Format(isoDate)
		cfg := getSettings()

		if cfg.ChatID != 0 {
			if hhmm == cfg.EveningReminder && lastEvening != today {
				lastEvening = today
				s.remindToday(cfg.ChatID, today)
			}
			if hhmm == cfg.AfternoonReminder && lastAfternoon != today {
				lastAfternoon = today
				s.remindYesterday(cfg.ChatID, now)
			}
		}
		time.Sleep(20 * time.Second)
	}
}

// remindToday nudges the user to fill today's Day form — unless it's already done.
func (s *server) remindToday(chatID int64, today string) {
	if dayFilled(today) {
		return
	}
	s.sendReminder(chatID, translate("remind_evening"), today)
}

// remindYesterday is the next-day catch-up: only if yesterday's Day form is
// missing, and it opens the form pre-set to yesterday.
func (s *server) remindYesterday(chatID int64, now time.Time) {
	yesterday := now.AddDate(0, 0, -1).Format(isoDate)
	if dayFilled(yesterday) {
		return
	}
	s.sendReminder(chatID, fmt.Sprintf(translate("remind_afternoon"), yesterday), yesterday)
}

func (s *server) sendReminder(chatID int64, text, targetDate string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if s.webAppURL != "" {
		msg.ReplyMarkup = s.dayKeyboard(targetDate)
	}
	s.send(msg)
}

// dayFilled reports whether the date's Day part is filled. On error it returns
// false — better to remind than to stay silent.
func dayFilled(date string) bool {
	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not check whether %s is filled: %v", date, err)
		return false
	}
	_, row := findDateRow(rows, date)
	return row != nil && partFilled(row, ownerDay)
}

// --- /settings, /evening, /afternoon ---

func settingsMessage() string {
	cfg := getSettings()
	return fmt.Sprintf(translate("settings_current"), cfg.EveningReminder, cfg.AfternoonReminder)
}

// setReminderTime validates an HH:MM argument and saves it for the given slot.
func setReminderTime(slot string, fields []string) string {
	if len(fields) < 2 {
		return translate("settings_usage")
	}
	value := fields[1]
	if _, err := time.Parse("15:04", value); err != nil {
		return translate("settings_bad_time")
	}
	err := saveSettings(func(cfg *settings) {
		if slot == "evening" {
			cfg.EveningReminder = value
		} else {
			cfg.AfternoonReminder = value
		}
	})
	if err != nil {
		log.Printf("could not save reminder time: %v", err)
		return translate("form_error")
	}
	return fmt.Sprintf(translate("settings_saved"), value)
}
