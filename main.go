package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/joho/godotenv"
)

// formAnswers mirrors the JSON the Mini App form sends back via tg.sendData
// (see docs/app.js). The json tags must match the keys used there.
type formAnswers struct {
	Mood   string `json:"mood"`
	Energy string `json:"energy"`
	Note   string `json:"note"`
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

	// Save: date first, then the three answers. appendRow leaves existing data intact.
	today := time.Now().Format("2006-01-02")
	if err := appendRow(today, answers.Mood, answers.Energy, answers.Note); err != nil {
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
