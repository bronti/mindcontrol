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
		"start":       "Hi! I'm Makhi-Bot 🤖\nTap the button below to open the daily form, or send /ping to check that I'm online.",
		"ping":        "pong 🏓 I'm alive!",
		"unknown":     "Got your message: %s\n(try /form to open the daily questions, or /ping)",
		"form_prompt": "Tap the button below to open the daily form 👇",
		"open_form":   "📝 Open form",
		"form_saved":  "Saved, thank you! ✅ See you tomorrow.",
		"form_error":  "Hmm, I couldn't save that 😞 Please try again in a moment.",
	},
	"ru": {
		"start":       "Привет! Я Makhi-Bot 🤖\nНажми кнопку ниже, чтобы открыть форму дня, или напиши /ping для проверки связи.",
		"ping":        "понг 🏓 я живой!",
		"unknown":     "Получил твоё сообщение: %s\n(нажми /form, чтобы открыть вопросы дня, или /ping)",
		"form_prompt": "Нажми кнопку ниже, чтобы открыть форму дня 👇",
		"open_form":   "📝 Открыть форму",
		"form_saved":  "Сохранил, спасибо! ✅ До завтра.",
		"form_error":  "Хм, не получилось сохранить 😞 Попробуй ещё раз чуть позже.",
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
