package main

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// runReminders is a small scheduler: every ~20 seconds it checks whether the
// current local time matches a configured reminder time, and fires at most once
// per day for each. It uses the machine's local clock, so reminders only happen
// while the bot (and this computer) are running.
func runReminders(bot *tgbotapi.BotAPI, webAppURL string) {
	var lastEvening, lastAfternoon string // the date each reminder last fired on

	for {
		now := time.Now()
		hhmm := now.Format("15:04")
		today := now.Format("2006-01-02")
		s := getSettings()

		if s.ChatID != 0 {
			if hhmm == s.EveningReminder && lastEvening != today {
				lastEvening = today
				sendEveningReminder(bot, webAppURL, s.ChatID, today)
			}
			if hhmm == s.AfternoonReminder && lastAfternoon != today {
				lastAfternoon = today
				sendAfternoonReminder(bot, webAppURL, s.ChatID, now)
			}
		}

		time.Sleep(20 * time.Second)
	}
}

// sendEveningReminder nudges the user to fill today — unless today is already done.
func sendEveningReminder(bot *tgbotapi.BotAPI, webAppURL string, chatID int64, today string) {
	if dateFilled(today) {
		return // already filled today — no nagging
	}
	sendReminder(bot, webAppURL, chatID, translate("remind_evening"), today)
}

// sendAfternoonReminder is the next-day catch-up: it only fires if yesterday has
// no entry, and opens the form pre-set to yesterday for a quick backfill.
func sendAfternoonReminder(bot *tgbotapi.BotAPI, webAppURL string, chatID int64, now time.Time) {
	yesterday := now.AddDate(0, 0, -1).Format("2006-01-02")
	if dateFilled(yesterday) {
		return // yesterday is filled — nothing to catch up on
	}
	sendReminder(bot, webAppURL, chatID, fmt.Sprintf(translate("remind_afternoon"), yesterday), yesterday)
}

// sendReminder sends a reminder message with the "Open form" button, optionally
// pre-selecting a date in the form (used for the yesterday catch-up).
func sendReminder(bot *tgbotapi.BotAPI, webAppURL string, chatID int64, text, targetDate string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if webAppURL != "" {
		msg.ReplyMarkup = formKeyboardForDate(webAppURL, targetDate)
	}
	if _, err := bot.Send(msg); err != nil {
		log.Printf("failed to send reminder: %v", err)
	}
}

// dateFilled reports whether the given date already has a row in the sheet.
// On error it returns false — better to remind than to stay silent.
func dateFilled(date string) bool {
	dates, err := existingDates()
	if err != nil {
		log.Printf("could not check whether %s is filled: %v", date, err)
		return false
	}
	for _, d := range dates {
		if d == date {
			return true
		}
	}
	return false
}

// settingsMessage shows the current reminder times and how to change them.
func settingsMessage() string {
	s := getSettings()
	return fmt.Sprintf(translate("settings_current"), s.EveningReminder, s.AfternoonReminder)
}

// setReminderTime validates an HH:MM argument and saves it for the given slot
// ("evening" or "afternoon"), returning the reply to send back.
func setReminderTime(which string, fields []string) string {
	if len(fields) < 2 {
		return translate("settings_usage")
	}
	value := fields[1]
	if _, err := time.Parse("15:04", value); err != nil {
		return translate("settings_bad_time")
	}
	err := saveSettings(func(s *settings) {
		if which == "evening" {
			s.EveningReminder = value
		} else {
			s.AfternoonReminder = value
		}
	})
	if err != nil {
		log.Printf("could not save reminder time: %v", err)
		return translate("form_error")
	}
	return fmt.Sprintf(translate("settings_saved"), value)
}
