package main

import "os"

// currentLanguage returns the selected interface language ("en" or "ru").
// Set it with BOT_LANGUAGE in .env (defaults to "en" if unset or unknown).
func currentLanguage() string {
	if os.Getenv("BOT_LANGUAGE") == "ru" {
		return "ru"
	}
	return "en"
}

// messages holds every user-facing string, grouped by language.
// To add a language, add a new map with the same keys.
var messages = map[string]map[string]string{
	"en": {
		"start":             "Hi! I'm Makhi-Bot 🤖\nTap the button below to open the daily form, or use /settings to set reminder times.",
		"ping":              "pong 🏓 I'm alive!",
		"unknown":           "Got your message: %s\n(try /form to open the daily questions, or /settings)",
		"form_prompt":       "Two forms: 🌙 Sleep and ☀️ Day. Tap a button below 👇",
		"open_form":         "📝 Open form",
		"open_sleep":        "🌙 Sleep",
		"open_day":          "☀️ Day",
		"open_calendar":     "📅 Calendar",
		"form_saved":        "Saved, thank you! ✅ See you tomorrow.",
		"saved_sleep":       "Sleep saved ✅",
		"saved_day":         "Day saved ✅ See you tomorrow.",
		"form_error":        "Hmm, I couldn't save that 😞 Please try again in a moment.",
		"date_taken":        "The day %s is already filled in — pick another date.",
		"taken_sleep":       "Sleep for %s is already filled in.",
		"taken_day":         "The day %s is already filled in.",
		"part_sleep":        "sleep",
		"part_day":          "day",
		"edit_prompt":       "✏️ Editing the %s of %s — tap the button to open it.",
		"remind_evening":    "🌙 Time for today's check-in! Tap the button to fill it in.",
		"remind_afternoon":  "📅 Yesterday (%s) isn't filled in yet — want to add it now?",
		"settings_current":  "⏰ Reminders (this computer's local time):\n• Evening: %s\n• Next-day catch-up (only if the day before is missing): %s\n\nChange them with:\n/evening HH:MM\n/afternoon HH:MM",
		"settings_usage":    "Usage: /evening HH:MM  (for example: /evening 21:00)",
		"settings_bad_time": "That doesn't look like a time. Use HH:MM, e.g. 21:00.",
		"settings_saved":    "Saved ✅ New time: %s",
	},
	"ru": {
		"start":             "Привет! Я Makhi-Bot 🤖\nНажми кнопку ниже, чтобы открыть форму дня, или /settings — настроить напоминания.",
		"ping":              "понг 🏓 я живой!",
		"unknown":           "Получил твоё сообщение: %s\n(нажми /form, чтобы открыть вопросы дня, или /settings)",
		"form_prompt":       "Две формы: 🌙 Сон и ☀️ День. Нажми кнопку ниже 👇",
		"open_form":         "📝 Открыть форму",
		"open_sleep":        "🌙 Сон",
		"open_day":          "☀️ День",
		"open_calendar":     "📅 Календарь",
		"form_saved":        "Сохранил, спасибо! ✅ До завтра.",
		"saved_sleep":       "Сон сохранён ✅",
		"saved_day":         "День сохранён ✅ До завтра.",
		"form_error":        "Хм, не получилось сохранить 😞 Попробуй ещё раз чуть позже.",
		"date_taken":        "День %s уже заполнен — выбери другую дату.",
		"taken_sleep":       "Сон за %s уже заполнен.",
		"taken_day":         "День %s уже заполнен.",
		"part_sleep":        "сон",
		"part_day":          "день",
		"edit_prompt":       "✏️ Редактирование (%s) за %s — нажми кнопку, чтобы открыть.",
		"remind_evening":    "🌙 Пора заполнить сегодняшний чек-ин! Нажми кнопку ниже.",
		"remind_afternoon":  "📅 Вчерашний день (%s) ещё не заполнен — добавить сейчас?",
		"settings_current":  "⏰ Напоминания (по локальному времени этого компьютера):\n• Вечернее: %s\n• Догоняющее на следующий день (только если предыдущий день пуст): %s\n\nИзменить:\n/evening ЧЧ:ММ\n/afternoon ЧЧ:ММ",
		"settings_usage":    "Формат: /evening ЧЧ:ММ  (например: /evening 21:00)",
		"settings_bad_time": "Это не похоже на время. Используй ЧЧ:ММ, напр. 21:00.",
		"settings_saved":    "Сохранил ✅ Новое время: %s",
	},
}

// translate returns the string for the given key in the current language,
// falling back to English if the key is missing in that language.
func translate(key string) string {
	if s, ok := messages[currentLanguage()][key]; ok {
		return s
	}
	return messages["en"][key]
}
