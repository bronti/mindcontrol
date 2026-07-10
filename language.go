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
		"start":                  "Hi! I'm Makhi-Bot 🤖\nTap the button below to open the daily form, or use /settings to set reminder times.",
		"ping":                   "pong 🏓 I'm alive!",
		"unknown":                "Got your message: %s\n(try /form to open the daily questions, or /settings)",
		"form_prompt":            "Two forms: 🌙 Sleep and ☀️ Day. Tap a button below 👇",
		"open_sleep":             "🌙 Sleep",
		"open_day":               "☀️ Day",
		"open_calendar":          "📅 Calendar",
		"saved_sleep":            "Sleep saved ✅",
		"saved_day":              "Day saved ✅ See you tomorrow.",
		"form_error":             "Hmm, I couldn't save that 😞 Please try again in a moment.",
		"taken_sleep":            "Sleep for %s is already filled in.",
		"taken_day":              "The day %s is already filled in.",
		"part_sleep":             "sleep",
		"part_day":               "day",
		"edit_prompt":            "✏️ Editing the %s of %s — tap the button to open it.",
		"remind_evening":         "🌙 Time for today's check-in! Tap the button to fill it in.",
		"remind_afternoon":       "📅 Yesterday (%s) isn't filled in yet — want to add it now?",
		"settings_current":       "⏰ Reminders (this computer's local time):\n• Evening: %s\n• Next-day catch-up (only if the day before is missing): %s\n\nChange them with:\n/evening HH:MM\n/afternoon HH:MM",
		"settings_usage":         "Usage: /evening HH:MM  (for example: /evening 21:00)",
		"settings_bad_time":      "That doesn't look like a time. Use HH:MM, e.g. 21:00.",
		"settings_saved":         "Saved ✅ New time: %s",
		"not_authorized":         "⛔ Sorry, this is a private bot — it only works for its owner.",
		"header_mismatch":        "⚠️ The sheet's header row no longer matches my columns, so I've paused — I won't save anything or send reminders until it's fixed (otherwise data would land in the wrong columns).\n\nPlease set the Makhi-Bot tab's header (row 1) to exactly this, then tap the button:\n\n%s",
		"header_fix_button":      "✅ The table is fixed",
		"header_synced":          "✅ Header updated — I'm working again.",
		"header_paused":          "⏸ I'm paused until the header is fixed. Update row 1 of the Makhi-Bot tab, then tap “✅ The table is fixed”.",
		"header_sync_failed":     "could not sync the header row: %v",
		"header_mismatch_log":    "sheet header differs from the schema — pausing until the owner fixes the table",
		"header_fix_sync_failed": "could not sync the header after a fix: %v",
		"header_resumed_log":     "header synced after a manual fix — resuming normal work",
		"header_no_chat_log":     "header mismatch, but there's no chat to notify yet — message the bot once, then fix the header and restart",
	},
	"ru": {
		"start":                  "Привет! Я Makhi-Bot 🤖\nНажми кнопку ниже, чтобы открыть форму дня, или /settings — настроить напоминания.",
		"ping":                   "понг 🏓 я живой!",
		"unknown":                "Получил твоё сообщение: %s\n(нажми /form, чтобы открыть вопросы дня, или /settings)",
		"form_prompt":            "Две формы: 🌙 Сон и ☀️ День. Нажми кнопку ниже 👇",
		"open_sleep":             "🌙 Сон",
		"open_day":               "☀️ День",
		"open_calendar":          "📅 Календарь",
		"saved_sleep":            "Сон сохранён ✅",
		"saved_day":              "День сохранён ✅ До завтра.",
		"form_error":             "Хм, не получилось сохранить 😞 Попробуй ещё раз чуть позже.",
		"taken_sleep":            "Сон за %s уже заполнен.",
		"taken_day":              "День %s уже заполнен.",
		"part_sleep":             "сон",
		"part_day":               "день",
		"edit_prompt":            "✏️ Редактирование (%s) за %s — нажми кнопку, чтобы открыть.",
		"remind_evening":         "🌙 Пора заполнить сегодняшний чек-ин! Нажми кнопку ниже.",
		"remind_afternoon":       "📅 Вчерашний день (%s) ещё не заполнен — добавить сейчас?",
		"settings_current":       "⏰ Напоминания (по локальному времени этого компьютера):\n• Вечернее: %s\n• Догоняющее на следующий день (только если предыдущий день пуст): %s\n\nИзменить:\n/evening ЧЧ:ММ\n/afternoon ЧЧ:ММ",
		"settings_usage":         "Формат: /evening ЧЧ:ММ  (например: /evening 21:00)",
		"settings_bad_time":      "Это не похоже на время. Используй ЧЧ:ММ, напр. 21:00.",
		"settings_saved":         "Сохранил ✅ Новое время: %s",
		"not_authorized":         "⛔ Извините, это личный бот — он работает только для владельца.",
		"header_mismatch":        "⚠️ Заголовок таблицы больше не совпадает с моими колонками, поэтому я поставил себя на паузу — ничего не сохраняю и не шлю напоминания, пока это не исправлено (иначе данные попадут не в те колонки).\n\nСделай строку заголовков (строка 1) на вкладке Makhi-Bot точно такой, затем нажми кнопку:\n\n%s",
		"header_fix_button":      "✅ Таблица исправлена",
		"header_synced":          "✅ Заголовок обновлён — снова работаю.",
		"header_paused":          "⏸ Я на паузе, пока не исправлен заголовок. Обнови строку 1 на вкладке Makhi-Bot и нажми «✅ Таблица исправлена».",
		"header_sync_failed":     "не удалось синхронизировать строку заголовков: %v",
		"header_mismatch_log":    "заголовок таблицы отличается от схемы — пауза, пока владелец не исправит таблицу",
		"header_fix_sync_failed": "не удалось синхронизировать заголовок после исправления: %v",
		"header_resumed_log":     "заголовок синхронизирован после ручного исправления — возобновляю обычную работу",
		"header_no_chat_log":     "заголовок не совпадает, но пока некому отправить сообщение — напиши боту один раз, затем исправь заголовок и перезапусти",
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
