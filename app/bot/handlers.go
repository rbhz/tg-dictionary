package bot

import (
	"context"

	"github.com/rbhz/tg-dictionary/app/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	callbackIdQuizReply = "qr"
	callbackIdSettings  = "st"
)

type Bot interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
	SendCallback(tgbotapi.CallbackConfig) (*tgbotapi.APIResponse, error)
	DB() db.Storage
}

func addToDictHandler(ctx context.Context, b *TelegramBot, u tgbotapi.Update) {
}
