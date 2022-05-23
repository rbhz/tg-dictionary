package bot

import (
	"context"
	"time"

	"github.com/rbhz/tg-dictionary/app/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type Handler interface {
	Handle(ctx context.Context, b Bot, u tgbotapi.Update)
	Passthrough(tgbotapi.Update) bool
	Match(u tgbotapi.Update) bool
}

// TelegramBot handles Telegram API intragration and updates handling
type TelegramBot struct {
	UserName string
	api      *tgbotapi.BotAPI
	db       db.Storage
	handlers []Handler
}

func (b *TelegramBot) processUpdate(u tgbotapi.Update) {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()
	tgUser := u.SentFrom()
	if tgUser != nil {
		var user db.User
		user, err := b.db.GetUser(db.UserID(tgUser.ID))
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				user = db.User{
					ID:       db.UserID(tgUser.ID),
					Username: tgUser.UserName,
					IsAdmin:  false,
					Language: tgUser.LanguageCode,
				}
				if err := b.db.SaveUser(user); err != nil {
					log.Error().Err(err).Int64("user", tgUser.ID).Msg("failed to save user")
				}
			} else {
				log.Error().Err(err).Int64("user", tgUser.ID).Msg("failed to get user")
				return
			}
		}
		ctx = context.WithValue(ctx, "user", user)
	}
	for _, handler := range b.handlers {
		if handler.Match(u) {
			handler.Handle(ctx, b, u)
			if !handler.Passthrough(u) {
				break
			}
		}
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

func (b *TelegramBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	message, err := b.api.Send(c)
	if err != nil {
		log.Error().Err(err).Msg("failed to send")
	}
	return message, err
}

func (b *TelegramBot) DB() db.Storage {
	return b.db
}

func NewTelegramBot(token string, db db.Storage, handlers []Handler) (*TelegramBot, error) {
	botAPI, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize bot")
	}
	log.Info().Str("username", botAPI.Self.UserName).Msg("telegram bot initialized")
	return &TelegramBot{
		UserName: botAPI.Self.UserName,
		api:      botAPI,
		db:       db,
		handlers: handlers,
	}, nil
}
