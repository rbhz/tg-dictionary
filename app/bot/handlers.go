package bot

import (
	"github.com/rbhz/tg-dictionary/app/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	callbackIDQuizReply = "qr"
	callbackIDSettings  = "st"
)

// Bot describes bot for handlers
type Bot interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
	SendCallback(tgbotapi.CallbackConfig) (*tgbotapi.APIResponse, error)
	DB() db.Storage
}

// neverPassthorugh implements Passthrough with always false
type neverPassthorugh struct{}

// Passthrough always returns false
func (h neverPassthorugh) Passthrough(u tgbotapi.Update) bool {
	return false
}
