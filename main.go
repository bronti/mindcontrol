package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	Bedtime      string       `json:"bedtime"`
	Wake         string       `json:"wake"`
	SleepHours   *float64     `json:"sleep_hours"`
	SleepQuality int          `json:"sleep_quality"`
	Dreams       string       `json:"dreams"`
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

// row lays the answers out as one spreadsheet row, in a fixed column order.
// Keep this order stable — it's the layout of the Makhi-Bot tab. If you add a
// question, add its column at the END so existing rows stay aligned.
func (a formAnswers) row(date string) []interface{} {
	return []interface{}{
		date,                             // A  date
		a.Bedtime,                        // B  fell asleep at
		a.Wake,                           // C  woke up at
		sleepCell(a.SleepHours),          // D  sleep duration (hours)
		a.SleepQuality,                   // E  sleep quality 1–10
		a.Dreams,                         // F  none / dreams / nightmares
		a.State,                          // G  overall state 1–10
		a.Anxiety,                        // H
		a.Irritability,                   // I
		a.Libido,                         // J
		a.Drowsiness,                     // K
		a.Appetite,                       // L
		a.Energy,                         // M
		a.AteWell,                        // N
		yesNo(a.Menstruation),            // O
		yesNo(a.Sex),                     // P
		yesNo(a.Masturbation),            // Q
		yesNo(a.Headache),                // R
		formatMedications(a.Medications), // S  e.g. "Lamotrigine 100mg; Fluoxetine 20mg"
		a.Note,                           // T  diary entry
	}
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

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("could not connect to the bot (check the token): %v", err)
	}
	log.Printf("Bot started: @%s. Send it /ping in Telegram", bot.Self.UserName)

	// Start receiving messages (long polling)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue // we only care about messages
		}

		// The form was submitted: Telegram delivers its JSON as web_app_data.
		if update.Message.WebAppData != nil {
			handleFormSubmission(bot, update.Message)
			continue
		}

		text := update.Message.Text
		log.Printf("[@%s] %s", update.Message.From.UserName, text)

		var reply string
		switch text {
		case "/start":
			reply = translate("start")
		case "/ping":
			reply = translate("ping")
		case "/form":
			reply = translate("form_prompt")
		default:
			reply = fmt.Sprintf(translate("unknown"), text)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyParameters = tgbotapi.ReplyParameters{MessageID: update.Message.MessageID}

		// On /start and /form, attach the reply keyboard with the "Open form" button.
		if (text == "/start" || text == "/form") && webAppURL != "" {
			msg.ReplyMarkup = formKeyboard(webAppURL)
		}

		if _, err := bot.Send(msg); err != nil {
			log.Printf("failed to send reply: %v", err)
		}
	}
}

// formKeyboard builds a reply keyboard (shown above the text input) with a
// single button that opens the Mini App form. Only a reply-keyboard button can
// send answers straight back to the bot via tg.sendData → WebAppData.
func formKeyboard(url string) tgbotapi.ReplyKeyboardMarkup {
	button := tgbotapi.NewKeyboardButtonWebApp(translate("open_form"), tgbotapi.WebAppInfo{URL: url})
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(button))
}

// handleFormSubmission parses the form's JSON, appends it to the sheet, and
// replies with a confirmation (or an error message if something went wrong).
func handleFormSubmission(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	raw := message.WebAppData.Data
	log.Printf("[@%s] form submitted: %s", message.From.UserName, raw)

	var answers formAnswers
	if err := json.Unmarshal([]byte(raw), &answers); err != nil {
		log.Printf("could not parse form data: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	// Save one row for today. appendRow leaves existing data intact.
	today := time.Now().Format("2006-01-02")
	if err := appendRow(answers.row(today)...); err != nil {
		log.Printf("could not save answers to the sheet: %v", err)
		reply(bot, message.Chat.ID, translate("form_error"))
		return
	}

	reply(bot, message.Chat.ID, translate("form_saved"))
}

// reply sends a plain text message to a chat (small helper to cut repetition).
func reply(bot *tgbotapi.BotAPI, chatID int64, text string) {
	if _, err := bot.Send(tgbotapi.NewMessage(chatID, text)); err != nil {
		log.Printf("failed to send reply: %v", err)
	}
}
