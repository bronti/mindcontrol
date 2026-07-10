package main

import (
	"log"
	"os"
	"strconv"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
	"github.com/joho/godotenv"
)

// Configuration constants — the bot's tunable text values, gathered here in one
// place. Change a value and rebuild; nothing else hard-codes these strings.
const (
	// sheetTab is the tab (worksheet) the bot reads and writes in the spreadsheet.
	// Every Sheets range is built from it (see tabRange in sheets.go); it's
	// single-quoted in ranges because of the hyphen. Change it if you rename the tab.
	sheetTab = "Makhi-Bot"

	// spreadsheetID identifies the Google Sheet (from its URL, between /d/ and /edit).
	spreadsheetID = "1bpCNYzsXwgHFLL4ylm3g3Smsb140kMUYKx2zcViEZAw"

	// settingsFile holds the bot's local runtime state — the reminder chat id and
	// times. It's gitignored (the chat id is private).
	settingsFile = "settings.json"

	// Default reminder times (HH:MM, in the configured time zone), used until the
	// owner changes them via /evening and /afternoon.
	defaultEveningReminder   = "21:00"
	defaultAfternoonReminder = "14:00"

	// Go reference-time layouts (see the time package) for formatting and parsing.
	isoDate         = "2006-01-02"          // date-only; sheet dates and form dates
	clockLayout     = "15:04"               // HH:MM; reminder times
	timestampLayout = "2006-01-02 15:04:05" // the "Last modified" cell
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

	ownerID, _ := strconv.ParseInt(os.Getenv("OWNER_ID"), 10, 64)
	if ownerID == 0 {
		log.Print("OWNER_ID is not set — the bot is OPEN to anyone. Send it a message to see " +
			"your user id in the log, then set OWNER_ID in .env and restart to lock it to you.")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Fatalf("could not connect to the bot (check the token): %v", err)
	}
	log.Printf("Bot started: @%s", bot.Self.UserName)

	loadSettings()
	srv := &server{
		bot:       bot,
		webAppURL: webAppURL,
		ownerID:   ownerID,
		// The medication list is personal, so it lives in .env — never in the
		// (public) form page. Same format as the sheet cells: "Name 200mg; Other".
		medications: os.Getenv("MEDICATIONS"),
		// Reminders and dates follow this time zone, not the host clock (a cloud
		// VM is usually UTC). See loadLocation.
		location: loadLocation(os.Getenv("TIMEZONE")),
	}

	// Sync the header row into row 1 — but if the sheet already has a *different*
	// non-empty header, don't overwrite it. Instead the bot messages the owner the
	// expected header and pauses until they fix the table and press the button.
	if err := srv.syncOrPauseForHeader(); err != nil {
		log.Printf("could not sync the header row: %v", err)
	}

	go srv.runReminders()

	updates := tgbotapi.NewUpdate(0)
	updates.Timeout = 60
	for update := range bot.GetUpdatesChan(updates) {
		srv.handleUpdate(update)
	}
}
