package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/joho/godotenv"
)

// medication is one entry from the form's medication list: a name and a dose
// in milligrams (kept as a string — it may be empty or a decimal like "12.5").
type medication struct {
	Name string `json:"name"`
	Dose string `json:"dose"`
}

// formAnswers mirrors the JSON the Mini App form sends back via tg.sendData
// (see docs/app.js). The json tags must match the keys used there.
// SleepHours is a pointer so a missing time (null) stays distinct from 0.
type formAnswers struct {
	FormType         string       `json:"form_type"` // "sleep" or "day" — which half of the day this is
	Date             string       `json:"date"`      // the day being filled in (chosen in the form)
	FilledAt         string       `json:"-"`         // when the form was submitted; set by the bot
	Bedtime          string       `json:"bedtime"`
	Wake             string       `json:"wake"`
	SleepHours       *float64     `json:"sleep_hours"`
	SleepQuality     *int         `json:"sleep_quality"`
	Dreams           string       `json:"dreams"`
	DreamNote        string       `json:"dream_note"`
	SleepMedications []medication `json:"sleep_medications"`
	State            *int         `json:"state"`
	Anxiety          *int         `json:"anxiety"`
	Irritability     *int         `json:"irritability"`
	Libido           *int         `json:"libido"`
	Drowsiness       *int         `json:"drowsiness"`
	Appetite         *int         `json:"appetite"`
	Energy           *int         `json:"energy"`
	AteWell          *int         `json:"ate_well"`
	Menstruation     bool         `json:"menstruation"`
	Sex              bool         `json:"sex"`
	Masturbation     bool         `json:"masturbation"`
	Headache         bool         `json:"headache"`
	Smoking          bool         `json:"smoking"`
	Medications      []medication `json:"medications"`
	Note             string       `json:"note"`
}

// Column ownership: which form fills a column. "meta" columns (date, filled-at)
// are written on every submission; "sleep"/"day" columns only by their own form.
const (
	ownerMeta  = "meta"
	ownerSleep = "sleep"
	ownerDay   = "day"
)

// columns is the single source of truth for the Makhi-Bot tab layout: the order
// here defines BOTH the header row and every data row. Grouped as
// date | sleep… | day… | filled-at. Reorder ONLY while the tab is empty — a
// reorder misaligns existing rows against the header.
var columns = []struct {
	header string
	owner  string
	value  func(a formAnswers) interface{}
}{
	{"Date", ownerMeta, func(a formAnswers) interface{} { return a.Date }},
	{"Fell asleep", ownerSleep, func(a formAnswers) interface{} { return a.Bedtime }},
	{"Woke up", ownerSleep, func(a formAnswers) interface{} { return a.Wake }},
	{"Sleep hours", ownerSleep, func(a formAnswers) interface{} { return sleepCell(a.SleepHours) }},
	{"How rested", ownerSleep, func(a formAnswers) interface{} { return numCell(a.SleepQuality) }},
	{"Dreams", ownerSleep, func(a formAnswers) interface{} { return a.Dreams }},
	{"Dream notes", ownerSleep, func(a formAnswers) interface{} { return dreamNote(a) }},
	{"Sleep medications", ownerSleep, func(a formAnswers) interface{} { return formatMedications(a.SleepMedications) }},
	{"Overall state", ownerDay, func(a formAnswers) interface{} { return numCell(a.State) }},
	{"Anxiety", ownerDay, func(a formAnswers) interface{} { return numCell(a.Anxiety) }},
	{"Irritability", ownerDay, func(a formAnswers) interface{} { return numCell(a.Irritability) }},
	{"Libido", ownerDay, func(a formAnswers) interface{} { return numCell(a.Libido) }},
	{"Drowsiness", ownerDay, func(a formAnswers) interface{} { return numCell(a.Drowsiness) }},
	{"Appetite", ownerDay, func(a formAnswers) interface{} { return numCell(a.Appetite) }},
	{"Energy", ownerDay, func(a formAnswers) interface{} { return numCell(a.Energy) }},
	{"Ate well", ownerDay, func(a formAnswers) interface{} { return numCell(a.AteWell) }},
	{"Menstruation", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Menstruation) }},
	{"Sex", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Sex) }},
	{"Masturbation", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Masturbation) }},
	{"Headache", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Headache) }},
	{"Smoking", ownerDay, func(a formAnswers) interface{} { return yesNo(a.Smoking) }},
	{"Medications", ownerDay, func(a formAnswers) interface{} { return formatMedications(a.Medications) }},
	{"Diary", ownerDay, func(a formAnswers) interface{} { return a.Note }},
	{"Filled at", ownerMeta, func(a formAnswers) interface{} { return a.FilledAt }},
}

// dreamNote returns the dream text only when there actually were dreams or
// nightmares — so text typed and then dismissed (dreams set back to "none")
// is never saved.
func dreamNote(a formAnswers) string {
	if a.Dreams == "dreams" || a.Dreams == "nightmares" || a.Dreams == "anxious" {
		return a.DreamNote
	}
	return ""
}

// headerRow returns the column headers, in schema order.
func headerRow() []interface{} {
	row := make([]interface{}, len(columns))
	for i, c := range columns {
		row[i] = c.header
	}
	return row
}

// mergeRow overlays one form's answers onto an existing row (pass nil existing
// for a brand-new day). Columns owned by the submitting part — and all meta
// columns — get fresh values; every other column keeps what was already there.
func mergeRow(existing []interface{}, a formAnswers, part string) []interface{} {
	row := make([]interface{}, len(columns))
	for i := range columns {
		if i < len(existing) {
			row[i] = existing[i]
		} else {
			row[i] = ""
		}
	}
	for i, c := range columns {
		if c.owner == part || c.owner == ownerMeta {
			row[i] = c.value(a)
		}
	}
	return row
}

// columnIndex returns the position of a column by its header, or -1 if absent.
func columnIndex(header string) int {
	for i, c := range columns {
		if c.header == header {
			return i
		}
	}
	return -1
}

// partFilled reports whether a row already has any value in the given part's columns.
func partFilled(row []interface{}, part string) bool {
	for i, c := range columns {
		if c.owner != part {
			continue
		}
		if i < len(row) && fmt.Sprint(row[i]) != "" {
			return true
		}
	}
	return false
}

// yesNo renders a boolean as a spreadsheet-friendly "yes"/"no".
func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// sleepCell renders the sleep duration, or "" if no times were entered.
func sleepCell(hours *float64) interface{} {
	if hours == nil {
		return ""
	}
	return *hours
}

// numCell renders a slider value, or "" if the slider was never touched (nil).
func numCell(n *int) interface{} {
	if n == nil {
		return ""
	}
	return *n
}

// formatMedications turns the medication list into a single readable cell.
func formatMedications(meds []medication) string {
	parts := make([]string, 0, len(meds))
	for _, m := range meds {
		if m.Dose != "" {
			parts = append(parts, fmt.Sprintf("%s %smg", m.Name, m.Dose))
		} else {
			parts = append(parts, m.Name)
		}
	}
	return strings.Join(parts, "; ")
}

func main() {
	// Read the bot token from the .env file (kept out of git — see .gitignore)
	_ = godotenv.Load()
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is not set — create a .env file with BOT_TOKEN=... (see .env.example)")
	}

	// URL of the Mini App form (served over HTTPS, e.g. GitHub Pages).
	// If empty, the bot still runs but can't show the "Open form" button.
	webAppURL := os.Getenv("WEB_APP_URL")
	if webAppURL == "" {
		log.Print("WEB_APP_URL is not set — the 'Open form' button will be hidden (see .env.example)")
	}

	// A per-startup version added to the form URL to bust the Mini App cache.
	formVersion = fmt.Sprint(time.Now().Unix())

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("could not connect to the bot (check the token): %v", err)
	}
	log.Printf("Bot started: @%s. Send it /ping in Telegram", bot.Self.UserName)

	// Keep the sheet's header row in step with the code (non-destructive).
	if err := syncHeader(headerRow()); err != nil {
		log.Printf("could not sync the header row: %v", err)
	}

	// Load local settings (chat id + reminder times) and start the reminder loop.
	loadSettings()
	go runReminders(bot, webAppURL)

	// Start receiving messages (long polling)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue // we only care about messages
		}

		// Remember which chat to send reminders to (learned from any message).
		rememberChat(update.Message.Chat.ID)

		// The form was submitted: Telegram delivers its JSON as web_app_data.
		if update.Message.WebAppData != nil {
			handleFormSubmission(bot, update.Message, webAppURL)
			continue
		}

		text := update.Message.Text
		log.Printf("[@%s] %s", update.Message.From.UserName, text)

		// The command is the first word, so "/evening 21:00" still matches "/evening".
		fields := strings.Fields(text)
		command := ""
		if len(fields) > 0 {
			command = fields[0]
		}

		var reply string
		switch command {
		case "/start":
			reply = translate("start")
		case "/ping":
			reply = translate("ping")
		case "/form":
			reply = translate("form_prompt")
		case "/settings":
			reply = settingsMessage()
		case "/evening":
			reply = setReminderTime("evening", fields)
		case "/afternoon":
			reply = setReminderTime("afternoon", fields)
		default:
			reply = fmt.Sprintf(translate("unknown"), text)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyParameters = tgbotapi.ReplyParameters{MessageID: update.Message.MessageID}

		// On /start and /form, attach the reply keyboard with the "Open form" button.
		if (command == "/start" || command == "/form") && webAppURL != "" {
			msg.ReplyMarkup = formKeyboard(webAppURL)
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("failed to send reply: %v", err)
		}
	}
}

// formVersion is set once at startup and added to the form URL as ?v=... It
// busts the Telegram Mini App's asset cache: restart the bot after changing the
// form and the new version forces the page (and app.js/style.css) to reload.
var formVersion string

// formKeyboard builds a reply keyboard with two buttons — one opens the Sleep
// form, one opens the Day form. Only a reply-keyboard button can send answers
// straight back to the bot via tg.sendData → WebAppData.
func formKeyboard(baseURL string) tgbotapi.ReplyKeyboardMarkup {
	sleepFilled, dayFilled, err := filledByPart()
	if err != nil {
		log.Printf("could not read filled dates for the form links: %v", err)
	}
	sleepBtn := tgbotapi.NewKeyboardButtonWebApp(translate("open_sleep"),
		tgbotapi.WebAppInfo{URL: buildFormURL(baseURL, ownerSleep, "", sleepFilled)})
	dayBtn := tgbotapi.NewKeyboardButtonWebApp(translate("open_day"),
		tgbotapi.WebAppInfo{URL: buildFormURL(baseURL, ownerDay, "", dayFilled)})
	calBtn := tgbotapi.NewKeyboardButtonWebApp(translate("open_calendar"),
		tgbotapi.WebAppInfo{URL: calendarURL(baseURL)})
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(sleepBtn, dayBtn),
		tgbotapi.NewKeyboardButtonRow(calBtn),
	)
}

// calendarURL builds the URL for the calendar page (calendar.html), carrying the
// cache-buster version and the compact per-day data.
func calendarURL(baseURL string) string {
	data, err := calendarData()
	if err != nil {
		log.Printf("could not read calendar data: %v", err)
	}
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	u.Path += "calendar.html"
	q := u.Query()
	if formVersion != "" {
		q.Set("v", formVersion)
	}
	if data != "" {
		q.Set("days", data)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// dayKeyboard builds a keyboard with just the Day-form button, optionally
// pre-selecting a date (used by the catch-up reminder for yesterday).
func dayKeyboard(baseURL, targetDate string) tgbotapi.ReplyKeyboardMarkup {
	_, dayFilled, err := filledByPart()
	if err != nil {
		log.Printf("could not read filled dates for the form link: %v", err)
	}
	btn := tgbotapi.NewKeyboardButtonWebApp(translate("open_day"),
		tgbotapi.WebAppInfo{URL: buildFormURL(baseURL, ownerDay, targetDate, dayFilled)})
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(btn))
}

// buildFormURL builds the Mini App URL for one form: the cache-buster version,
// which form to show (?form=), an optional pre-selected date, and the dates whose
// matching part is already filled (so the form can grey them out). The filled
// list is capped to the most recent dates to keep the URL from growing forever.
func buildFormURL(baseURL, part, targetDate string, filled []string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	if formVersion != "" {
		q.Set("v", formVersion)
	}
	q.Set("form", part)
	if targetDate != "" {
		q.Set("date", targetDate)
	}
	if len(filled) > 0 {
		const maxDates = 120
		sorted := append([]string(nil), filled...)
		sort.Strings(sorted) // ISO dates sort chronologically
		if len(sorted) > maxDates {
			sorted = sorted[len(sorted)-maxDates:] // keep the most recent
		}
		q.Set("filled", strings.Join(sorted, ","))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// rememberChat saves the chat id the first time we see it (or if it changes),
// so the reminder loop knows where to send messages.
func rememberChat(chatID int64) {
	if getSettings().ChatID == chatID {
		return
	}
	if err := saveSettings(func(s *settings) { s.ChatID = chatID }); err != nil {
		log.Printf("could not save chat id: %v", err)
		return
	}
	log.Printf("reminders will be sent to chat %d", chatID)
}

// handleFormSubmission parses one form's JSON, validates it, and merges it into
// that day's row — creating the row if new, updating it if it already exists.
// Re-submitting an already-filled part is refused so nothing is overwritten.
func handleFormSubmission(bot *tgbotapi.BotAPI, message *tgbotapi.Message, webAppURL string) {
	raw := message.WebAppData.Data
	log.Printf("[@%s] form submitted: %s", message.From.UserName, raw)

	var a formAnswers
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		log.Printf("could not parse form data: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	if a.FormType != ownerSleep && a.FormType != ownerDay {
		log.Printf("unknown form_type: %q", a.FormType)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}
	// The date is chosen in the form — make sure it's a real YYYY-MM-DD.
	if _, err := time.Parse("2006-01-02", a.Date); err != nil {
		log.Printf("bad date from form: %q (%v)", a.Date, err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	rowNum, existing, err := findDateRow(a.Date)
	if err != nil {
		log.Printf("could not look up the date row: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// This part is already filled for that day — don't overwrite it.
	if rowNum != 0 && partFilled(existing, a.FormType) {
		reply(bot, message.Chat.ID, fmt.Sprintf(translate("taken_"+a.FormType), a.Date))
		return
	}

	// Stamp the submission time and merge this part into the day's row.
	a.FilledAt = time.Now().Format("2006-01-02 15:04:05")
	merged := mergeRow(existing, a, a.FormType)
	if rowNum != 0 {
		err = updateRow(rowNum, merged)
	} else {
		err = appendRow(merged...)
	}
	if err != nil {
		log.Printf("could not save the row: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// Confirm, and offer fresh buttons so the filled-out days update.
	msg := tgbotapi.NewMessage(message.Chat.ID, translate("saved_"+a.FormType))
	if webAppURL != "" {
		msg.ReplyMarkup = formKeyboard(webAppURL)
	}
	if _, err := bot.Send(msg); err != nil {
		log.Printf("failed to send confirmation: %v", err)
	}
}

// reply sends a plain text message to a chat (small helper to cut repetition).
func reply(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if _, err := bot.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("failed to send reply: %v", err)
	}
}
