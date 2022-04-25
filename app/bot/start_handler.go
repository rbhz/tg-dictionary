package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type StartHandler struct{}

func (h StartHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "start"
}

func (h StartHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h StartHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Hi! Just send me a word!"))
}
