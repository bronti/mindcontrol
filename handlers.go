package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// server bundles the bot and its Mini App URL so handlers and keyboards don't
// have to thread them through every call.
type server struct {
	bot       *tgbotapi.BotAPI
	webAppURL string
	ownerID   int64 // Telegram user id allowed to use the bot; 0 = open (setup mode)
}

const isoDate = "2006-01-02"

// --- routing ---

func (s *server) handleUpdate(update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	if !s.authorized(update.Message) {
		return
	}
	s.rememberChat(update.Message.Chat.ID)

	if update.Message.WebAppData != nil {
		s.handleWebAppData(update.Message)
		return
	}
	s.handleCommand(update.Message)
}

func (s *server) handleCommand(message *tgbotapi.Message) {
	text := message.Text
	log.Printf("[@%s] %s", message.From.UserName, text)

	// The command is the first word, so "/evening 21:00" still matches "/evening".
	fields := strings.Fields(text)
	command := ""
	if len(fields) > 0 {
		command = fields[0]
	}

	var answer string
	switch command {
	case "/start":
		answer = translate("start")
	case "/ping":
		answer = translate("ping")
	case "/form":
		answer = translate("form_prompt")
	case "/settings":
		answer = settingsMessage()
	case "/evening":
		answer = setReminderTime("evening", fields)
	case "/afternoon":
		answer = setReminderTime("afternoon", fields)
	default:
		answer = fmt.Sprintf(translate("unknown"), text)
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, answer)
	msg.ReplyParameters = tgbotapi.ReplyParameters{MessageID: message.MessageID}
	if (command == "/start" || command == "/form") && s.webAppURL != "" {
		msg.ReplyMarkup = s.formKeyboard()
	}
	s.send(msg)
}

// handleWebAppData routes what the Mini App sent: a calendar "edit" request or a
// normal form submission.
func (s *server) handleWebAppData(message *tgbotapi.Message) {
	raw := message.WebAppData.Data
	var probe struct {
		T string `json:"t"`
	}
	_ = json.Unmarshal([]byte(raw), &probe)
	if probe.T == "edit" {
		s.handleEditRequest(message, raw)
		return
	}
	s.handleFormSubmission(message)
}

// --- edit requests (from tapping a day in the calendar) ---

func (s *server) handleEditRequest(message *tgbotapi.Message, raw string) {
	log.Printf("[@%s] edit request: %s", message.From.UserName, raw)

	var req struct {
		Date string `json:"date"`
		Part string `json:"part"`
	}
	if err := json.Unmarshal([]byte(raw), &req); err != nil ||
		(req.Part != ownerSleep && req.Part != ownerDay) || s.webAppURL == "" {
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}
	if _, err := time.Parse(isoDate, req.Date); err != nil {
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}

	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not read the sheet for the edit request: %v", err)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}
	_, row := findDateRow(rows, req.Date) // row is nil when the day has no entry yet

	msg := tgbotapi.NewMessage(message.Chat.ID,
		fmt.Sprintf(translate("edit_prompt"), translate("part_"+req.Part), req.Date))
	msg.ReplyMarkup = s.editKeyboard(req.Part, req.Date, row, latestMedicationsRows(rows, req.Part, req.Date))
	s.send(msg)
}

// --- form submissions (save or edit) ---

func (s *server) handleFormSubmission(message *tgbotapi.Message) {
	raw := message.WebAppData.Data
	log.Printf("[@%s] form submitted: %s", message.From.UserName, raw)

	var a formAnswers
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		log.Printf("could not parse form data: %v", err)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}
	if a.FormType != ownerSleep && a.FormType != ownerDay {
		log.Printf("unknown form_type: %q", a.FormType)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}
	if _, err := time.Parse(isoDate, a.Date); err != nil {
		log.Printf("bad date from form: %q (%v)", a.Date, err)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}

	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not read the sheet before saving: %v", err)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}
	rowNum, existing := findDateRow(rows, a.Date)

	// A normal submission won't overwrite an already-filled part; an edit will.
	if !a.Edit && rowNum != 0 && partFilled(existing, a.FormType) {
		s.reply(message.Chat.ID, fmt.Sprintf(translate("taken_"+a.FormType), a.Date))
		return
	}

	a.LastModified = time.Now().Format("2006-01-02 15:04:05")
	merged := mergeRow(existing, a, a.FormType)
	if rowNum != 0 {
		err = updateRow(rowNum, merged)
	} else {
		err = appendRow(merged...)
	}
	if err != nil {
		log.Printf("could not save the row: %v", err)
		s.reply(message.Chat.ID, translate("form_error"))
		return
	}

	// Confirm, and offer fresh buttons so the calendar's filled days stay current.
	msg := tgbotapi.NewMessage(message.Chat.ID, translate("saved_"+a.FormType))
	if s.webAppURL != "" {
		msg.ReplyMarkup = s.formKeyboard()
	}
	s.send(msg)
}

// rememberChat stores the chat id the first time we see it, so reminders know
// where to go.
func (s *server) rememberChat(chatID int64) {
	if getSettings().ChatID == chatID {
		return
	}
	if err := saveSettings(func(cfg *settings) { cfg.ChatID = chatID }); err != nil {
		log.Printf("could not save chat id: %v", err)
		return
	}
	log.Printf("reminders will be sent to chat %d", chatID)
}

// --- keyboards ---

// formKeyboard offers the two entry forms and the calendar. Only a reply-keyboard
// button can send answers back to the bot via tg.sendData → WebAppData.
func (s *server) formKeyboard() tgbotapi.ReplyKeyboardMarkup {
	// One read serves the whole keyboard; on error rows is nil and the helpers
	// degrade gracefully (no filled/pre-fill info, buttons still work).
	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not read sheet data for the keyboard: %v", err)
	}
	sleepDates, dayDates := filledByPartRows(rows)
	today := time.Now().Format(isoDate)
	sleepBtn := s.webAppButton("open_sleep", buildFormURL(s.webAppURL, ownerSleep, "", sleepDates, latestMedicationsRows(rows, ownerSleep, today)))
	dayBtn := s.webAppButton("open_day", buildFormURL(s.webAppURL, ownerDay, "", dayDates, latestMedicationsRows(rows, ownerDay, today)))
	calBtn := s.webAppButton("open_calendar", calendarURL(s.webAppURL, calendarDataRows(rows)))
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(sleepBtn, dayBtn),
		tgbotapi.NewKeyboardButtonRow(calBtn),
	)
}

// dayKeyboard is just the Day-form button, optionally pre-set to a date (used by
// the catch-up reminder).
func (s *server) dayKeyboard(targetDate string) tgbotapi.ReplyKeyboardMarkup {
	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not read sheet data for the keyboard: %v", err)
	}
	_, dayDates := filledByPartRows(rows)
	before := targetDate
	if before == "" {
		before = time.Now().Format(isoDate)
	}
	btn := s.webAppButton("open_day", buildFormURL(s.webAppURL, ownerDay, targetDate, dayDates, latestMedicationsRows(rows, ownerDay, before)))
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(btn))
}

// editKeyboard opens the form pre-filled to edit one day+part.
func (s *server) editKeyboard(part, date string, row []any, defaultMeds string) tgbotapi.ReplyKeyboardMarkup {
	labelKey := "open_sleep"
	if part == ownerDay {
		labelKey = "open_day"
	}
	btn := s.webAppButton(labelKey, buildEditURL(s.webAppURL, part, date, row, defaultMeds))
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(btn))
}

func (s *server) webAppButton(labelKey, link string) tgbotapi.KeyboardButton {
	return tgbotapi.NewKeyboardButtonWebApp(translate(labelKey), tgbotapi.WebAppInfo{URL: link})
}

// --- sending ---

func (s *server) reply(chatID int64, text string) {
	s.send(tgbotapi.NewMessage(chatID, text))
}

func (s *server) send(msg tgbotapi.MessageConfig) {
	if _, err := s.bot.Send(msg); err != nil {
		log.Printf("failed to send message: %v", err)
	}
}
