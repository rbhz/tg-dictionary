package bot

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// StartHandler is a handler for /start command
type StartHandler struct {
	neverPassthorugh
}

// Match returns true if update is /start command
func (h StartHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "start"
}

// Handle sends start message
func (h StartHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Hi! Just send me a word!"))
}
