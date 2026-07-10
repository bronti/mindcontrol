package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// server bundles the bot and its Mini App URL so handlers and keyboards don't
// have to thread them through every call.
type server struct {
	bot         *tgbotapi.BotAPI
	webAppURL   string
	ownerID     int64          // Telegram user id allowed to use the bot; 0 = open (setup mode)
	medications string         // the drugs the form offers to pick from (MEDICATIONS in .env)
	location    *time.Location // time zone for reminders and dates (TIMEZONE in .env)

	// paused is set when the sheet's header row differs from the schema at
	// startup: while paused the bot ignores forms and reminders (saving would
	// land data in the wrong columns) until the owner presses "the table is
	// fixed". Read/written from both the update loop and the reminders goroutine,
	// so it's atomic.
	paused atomic.Bool
}

// callbackTableFixed is the inline-button data for "the table is fixed" — the
// owner presses it after reconciling the sheet header, and the bot then rewrites
// row 1 and resumes normal work.
const callbackTableFixed = "table_fixed"

// --- routing ---

func (s *server) handleUpdate(update tgbotapi.Update) {
	// Inline-button presses (currently only "the table is fixed") arrive as
	// callback queries, not messages.
	if update.CallbackQuery != nil {
		s.handleCallback(update.CallbackQuery)
		return
	}
	if update.Message == nil {
		return
	}
	if !s.authorized(update.Message) {
		return
	}
	s.rememberChat(update.Message.Chat.ID)

	// While paused for a header mismatch, ignore normal traffic: the sheet's
	// columns don't line up with the schema, so saving would misplace data. The
	// bot only resumes when the owner presses "the table is fixed".
	if s.paused.Load() {
		s.reply(update.Message.Chat.ID, translate("header_paused"))
		return
	}

	if update.Message.WebAppData != nil {
		s.handleWebAppData(update.Message)
		return
	}
	s.handleCommand(update.Message)
}

// handleCallback handles inline-button presses. The only button is "the table is
// fixed": the owner presses it after reconciling the sheet header, and we then
// rewrite row 1 to the schema and lift the pause so normal work resumes.
func (s *server) handleCallback(cb *tgbotapi.CallbackQuery) {
	if s.ownerID != 0 && (cb.From == nil || cb.From.ID != s.ownerID) {
		return
	}
	if cb.Data != callbackTableFixed {
		return
	}
	// Acknowledge the tap so Telegram stops the button's loading spinner.
	_, _ = s.bot.Request(tgbotapi.NewCallback(cb.ID, ""))

	if !s.paused.Load() {
		return // already running — nothing to do
	}
	if err := syncHeader(headerRow()); err != nil {
		log.Printf("could not sync the header after a fix: %v", err)
		if cb.Message != nil {
			s.reply(cb.Message.Chat.ID, translate("form_error"))
		}
		return
	}
	s.paused.Store(false)
	log.Print("header synced after a manual fix — resuming normal work")
	if cb.Message != nil {
		s.reply(cb.Message.Chat.ID, translate("header_synced"))
	}
}

// syncOrPauseForHeader writes the schema's header into row 1 at startup — unless
// the sheet already has a DIFFERENT non-empty header. In that case it doesn't
// overwrite: it pauses the bot and asks the owner to reconcile the table first
// (see pauseForHeaderMismatch). A read error is returned so the caller can log it;
// the header is then left untouched.
func (s *server) syncOrPauseForHeader() error {
	existing, err := readHeaderRow()
	if err != nil {
		return err
	}
	want := headerRow()
	if !headerEmpty(existing) && !headerEqual(existing, want) {
		log.Print("sheet header differs from the schema — pausing until the owner fixes the table")
		s.pauseForHeaderMismatch(want)
		return nil
	}
	return syncHeader(want)
}

// pauseForHeaderMismatch stops normal work and messages the owner the header the
// schema expects, with a "the table is fixed" button. The bot stays paused until
// that button is pressed (handled in handleCallback).
func (s *server) pauseForHeaderMismatch(newHeader []any) {
	s.paused.Store(true)

	// Prefer the remembered reminder chat; fall back to the owner id, which in a
	// private chat is also the chat id.
	chatID := getSettings().ChatID
	if chatID == 0 {
		chatID = s.ownerID
	}
	if chatID == 0 {
		log.Print("header mismatch, but there's no chat to notify yet — message the bot " +
			"once, then fix the header and restart")
		return
	}

	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf(translate("header_mismatch"), formatHeader(newHeader)))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(translate("header_fix_button"), callbackTableFixed),
		),
	)
	s.send(msg)
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

	a.LastModified = s.now().Format(timestampLayout)
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
	rows := readRowsOrLog() // one read serves all three buttons
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(s.formButton(rows, ownerSleep, ""), s.formButton(rows, ownerDay, "")),
		tgbotapi.NewKeyboardButtonRow(s.webAppButton("open_calendar", calendarURL(s.webAppURL, calendarDataRows(rows)))),
	)
}

// dayKeyboard is just the Day-form button, optionally pre-set to a date (used by
// the catch-up reminder).
func (s *server) dayKeyboard(targetDate string) tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(s.formButton(readRowsOrLog(), ownerDay, targetDate)))
}

// editKeyboard opens the form pre-filled to edit one day+part.
func (s *server) editKeyboard(part, date string, row []any, defaultMeds string) tgbotapi.ReplyKeyboardMarkup {
	btn := s.webAppButton(formLabelKey(part), buildEditURL(s.webAppURL, part, date, row, defaultMeds, s.medications))
	return tgbotapi.NewReplyKeyboard(tgbotapi.NewKeyboardButtonRow(btn))
}

// formButton builds one form-opening button from already-read rows: its URL
// carries the part's filled dates (to grey them out) and the most recent
// medications (to pre-fill). An empty targetDate means "opened for today".
func (s *server) formButton(rows [][]any, part, targetDate string) tgbotapi.KeyboardButton {
	sleepDates, dayDates := filledByPartRows(rows)
	filled := sleepDates
	if part == ownerDay {
		filled = dayDates
	}
	medsBefore := targetDate
	if medsBefore == "" {
		medsBefore = s.now().Format(isoDate)
	}
	link := buildFormURL(s.webAppURL, part, targetDate, filled, latestMedicationsRows(rows, part, medsBefore), s.medications)
	return s.webAppButton(formLabelKey(part), link)
}

// formLabelKey is the translation key for a part's button label.
func formLabelKey(part string) string {
	if part == ownerSleep {
		return "open_sleep"
	}
	return "open_day"
}

// readRowsOrLog reads the sheet, logging (not failing) on error — a keyboard
// built from nil rows still works, just without filled/pre-fill info.
func readRowsOrLog() [][]any {
	rows, err := readDataRows()
	if err != nil {
		log.Printf("could not read sheet data for the keyboard: %v", err)
	}
	return rows
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
