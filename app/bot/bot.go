package bot

import (
	"context"
	"tg-dictionary/app/db"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// TelegramBot handles Telegram API intragration and updates handling
type TelegramBot struct {
	UserName string
	api      *tgbotapi.BotAPI
	db       db.Storage
}

func (b *TelegramBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	message, err := b.api.Send(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to send")
	}
	return message, err
}

func (b *TelegramBot) processUpdate(u tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	ctx = context.WithValue(ctx, "test", 1)
	if u.Message == nil {
		return
	}
	if u.Message.Command() == "start" {
		startHandler(ctx, b, u)
	} else {
		addToDictHandler(ctx, b, u)
	}
}
func (b *TelegramBot) Start() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	for u := range updates {
		b.processUpdate(u)
	}
}

func NewTelegramBot(token string, db db.Storage) (*TelegramBot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize bot")
	}
	log.Info().Str("username", botAPI.Self.UserName).Msg("telegram bot initialized")
	return &TelegramBot{
		UserName: botAPI.Self.UserName,
		api:      botAPI,
		db:       db,
	}, nil
}
