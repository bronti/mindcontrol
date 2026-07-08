package main

import (
	"log"

	tgbotapi "github.com/OvyFlash/telegram-bot-api"
)

// authorized gates every incoming update to the bot's owner.
//
// The owner is identified by their Telegram user id (OWNER_ID in .env). When
// OWNER_ID is unset the bot stays open — "setup mode" — but logs each sender's
// id so the owner can discover their own, drop it into .env, and lock the bot
// down. Once OWNER_ID is set, anyone else gets a short warning and is ignored.
func (s *server) authorized(message *tgbotapi.Message) bool {
	from := message.From

	if s.ownerID == 0 {
		if from != nil {
			log.Printf("OWNER_ID not set — the bot is OPEN. Message from @%s (id %d). "+
				"Set OWNER_ID=%d in .env and restart to lock the bot to yourself.",
				from.UserName, from.ID, from.ID)
		}
		return true
	}

	if from != nil && from.ID == s.ownerID {
		return true
	}

	name, id := "", int64(0)
	if from != nil {
		name, id = from.UserName, from.ID
	}
	log.Printf("blocked message from @%s (id %d)", name, id)
	s.reply(message.Chat.ID, translate("not_authorized"))
	return false
}
