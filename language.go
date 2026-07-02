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
		"start":   "Hi! I'm Makhi-Bot 🤖\nSend /ping to check that I'm online.",
		"ping":    "pong 🏓 I'm alive!",
		"unknown": "Got your message: %s\n(for now I only understand /ping — more coming soon)",
	},
	"ru": {
		"start":   "Привет! Я Makhi-Bot 🤖\nНапиши /ping, чтобы проверить, что я на связи.",
		"ping":    "понг 🏓 я живой!",
		"unknown": "Получил твоё сообщение: %s\n(пока умею только /ping — остальное скоро)",
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
