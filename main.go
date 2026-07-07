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
	Date         string       `json:"date"` // the day being filled in (chosen in the form)
	FilledAt     string       `json:"-"`    // when the form was submitted; set by the bot
	Bedtime      string       `json:"bedtime"`
	Wake         string       `json:"wake"`
	SleepHours   *float64     `json:"sleep_hours"`
	SleepQuality int          `json:"sleep_quality"`
	Dreams       string       `json:"dreams"`
	DreamNote    string       `json:"dream_note"`
	State        int          `json:"state"`
	Anxiety      int          `json:"anxiety"`
	Irritability int          `json:"irritability"`
	Libido       int          `json:"libido"`
	Drowsiness   int          `json:"drowsiness"`
	Appetite     int          `json:"appetite"`
	Energy       int          `json:"energy"`
	AteWell      int          `json:"ate_well"`
	Menstruation bool         `json:"menstruation"`
	Sex          bool         `json:"sex"`
	Masturbation bool         `json:"masturbation"`
	Headache     bool         `json:"headache"`
	Medications  []medication `json:"medications"`
	Note         string       `json:"note"`
}

// columns is the single source of truth for the Makhi-Bot tab layout: the order
// here defines BOTH the header row and every data row, so the two can never
// drift apart. To add a question, add one entry AT THE END. Never reorder or
// delete an entry — that would misalign every existing row against the header.
var columns = []struct {
	header string
	value  func(a formAnswers) interface{}
}{
	{"Date", func(a formAnswers) interface{} { return a.Date }},
	{"Fell asleep", func(a formAnswers) interface{} { return a.Bedtime }},
	{"Woke up", func(a formAnswers) interface{} { return a.Wake }},
	{"Sleep hours", func(a formAnswers) interface{} { return sleepCell(a.SleepHours) }},
	{"Sleep quality", func(a formAnswers) interface{} { return a.SleepQuality }},
	{"Dreams", func(a formAnswers) interface{} { return a.Dreams }},
	{"Overall state", func(a formAnswers) interface{} { return a.State }},
	{"Anxiety", func(a formAnswers) interface{} { return a.Anxiety }},
	{"Irritability", func(a formAnswers) interface{} { return a.Irritability }},
	{"Libido", func(a formAnswers) interface{} { return a.Libido }},
	{"Drowsiness", func(a formAnswers) interface{} { return a.Drowsiness }},
	{"Appetite", func(a formAnswers) interface{} { return a.Appetite }},
	{"Energy", func(a formAnswers) interface{} { return a.Energy }},
	{"Ate well", func(a formAnswers) interface{} { return a.AteWell }},
	{"Menstruation", func(a formAnswers) interface{} { return yesNo(a.Menstruation) }},
	{"Sex", func(a formAnswers) interface{} { return yesNo(a.Sex) }},
	{"Masturbation", func(a formAnswers) interface{} { return yesNo(a.Masturbation) }},
	{"Headache", func(a formAnswers) interface{} { return yesNo(a.Headache) }},
	{"Medications", func(a formAnswers) interface{} { return formatMedications(a.Medications) }},
	{"Diary", func(a formAnswers) interface{} { return a.Note }},
	{"Filled at", func(a formAnswers) interface{} { return a.FilledAt }},
	{"Dream notes", func(a formAnswers) interface{} { return dreamNote(a) }},
}

// dreamNote returns the dream text only when there actually were dreams or
// nightmares — so text typed and then dismissed (dreams set back to "none")
// is never saved.
func dreamNote(a formAnswers) string {
	if a.Dreams == "dreams" || a.Dreams == "nightmares" {
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

// row lays the answers out as one spreadsheet row, in schema order — the same
// order as headerRow, so values always land under the right header.
func (a formAnswers) row() []interface{} {
	row := make([]interface{}, len(columns))
	for i, c := range columns {
		row[i] = c.value(a)
	}
	return row
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

	// Make sure the sheet has a header row (written once, only when the tab is empty).
	if err := ensureHeader(headerRow()); err != nil {
		log.Printf("could not check/write the header row: %v", err)
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

// formKeyboard builds a reply keyboard (shown above the text input) with a
// single button that opens the Mini App form. Only a reply-keyboard button can
// send answers straight back to the bot via tg.sendData → WebAppData.
func formKeyboard(baseURL string) tgbotapi.ReplyKeyboardMarkup {
	return formKeyboardForDate(baseURL, "")
}

// formKeyboardForDate is like formKeyboard but pre-selects a date in the form
// (used by the yesterday catch-up reminder). Pass "" for the default (today).
func formKeyboardForDate(baseURL, targetDate string) tgbotapi.ReplyKeyboardMarkup {
	dates, err := existingDates()
	if err != nil {
		log.Printf("could not read existing dates for the form link: %v", err)
	}
	link := buildFormURL(baseURL, dates, targetDate)
	button := tgbotapi.NewKeyboardButtonWebApp(translate("open_form"), tgbotapi.WebAppInfo{URL: link})
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(button))
}

// buildFormURL adds the cache-buster version, an optional pre-selected date, and
// the already-filled dates to the form URL. The "filled" list lets the form grey
// out days that already have a row; it's capped to the most recent dates to keep
// the URL from growing forever.
func buildFormURL(baseURL string, dates []string, targetDate string) string {
	u, err := url.Parse(baseURL)
	if err != nil {
		return baseURL
	}
	q := u.Query()
	if formVersion != "" {
		q.Set("v", formVersion)
	}
	if targetDate != "" {
		q.Set("date", targetDate)
	}
	if len(dates) > 0 {
		const maxDates = 120
		sorted := append([]string(nil), dates...)
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

// handleFormSubmission parses the form's JSON, checks the chosen date is valid
// and not already filled, appends the row, and confirms (or reports an error).
func handleFormSubmission(bot *tgbotapi.BotAPI, message *tgbotapi.Message, webAppURL string) {
	raw := message.WebAppData.Data
	log.Printf("[@%s] form submitted: %s", message.From.UserName, raw)

	var answers formAnswers
	if err := json.Unmarshal([]byte(raw), &answers); err != nil {
		log.Printf("could not parse form data: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// The date is chosen in the form — make sure it's a real YYYY-MM-DD.
	if _, err := time.Parse("2006-01-02", answers.Date); err != nil {
		log.Printf("bad date from form: %q (%v)", answers.Date, err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// Refuse to fill a day that already has a row. This is the authoritative
	// check — the form's red highlight can be out of date.
	dates, err := existingDates()
	if err != nil {
		log.Printf("could not check existing dates: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}
	for _, d := range dates {
		if d == answers.Date {
			reply(bot, message.Chat.ID, fmt.Sprintf(translate("date_taken"), answers.Date))
			return
		}
	}

	// Stamp when the form was actually submitted, then save.
	answers.FilledAt = time.Now().Format("2006-01-02 15:04:05")
	if err := appendRow(answers.row()...); err != nil {
		log.Printf("could not save answers to the sheet: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// Confirm, and offer a fresh button so the next backfill knows this date is taken.
	msg := tgbotapi.NewMessage(message.Chat.ID, translate("form_saved"))
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
