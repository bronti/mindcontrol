package main

import (
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
)

func main() {
	// Read the bot token from the .env file (kept out of git — see .gitignore)
	_ = godotenv.Load()
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is not set — create a .env file with BOT_TOKEN=... (see .env.example)")
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

		text := update.Message.Text
		log.Printf("[@%s] %s", update.Message.From.UserName, text)

		var reply string
		switch text {
		case "/start":
			reply = t("start")
		case "/ping":
			reply = t("ping")
		default:
			reply = fmt.Sprintf(t("unknown"), text)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		msg.ReplyToMessageID = update.Message.MessageID
		if _, err := bot.Send(msg); err != nil {
			log.Printf("failed to send reply: %v", err)
		}
	}
}
