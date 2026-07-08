package main

import (
	"log"
	"os"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // .env holds BOT_TOKEN / WEB_APP_URL / BOT_LANGUAGE (gitignored)

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		log.Fatal("BOT_TOKEN is not set — create a .env file with BOT_TOKEN=... (see .env.example)")
	}
	webAppURL := os.Getenv("WEB_APP_URL")
	if webAppURL == "" {
		log.Print("WEB_APP_URL is not set — the form buttons will be hidden (see .env.example)")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("could not connect to the bot (check the token): %v", err)
	}
	log.Printf("Bot started: @%s", bot.Self.UserName)

	if err := syncHeader(headerRow()); err != nil {
		log.Printf("could not sync the header row: %v", err)
	}

	loadSettings()
	srv := &server{bot: bot, webAppURL: webAppURL}
	go srv.runReminders()

	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 60
	for update := range bot.GetUpdatesChan(updates) {
		srv.handleUpdate(update)
	}
}
