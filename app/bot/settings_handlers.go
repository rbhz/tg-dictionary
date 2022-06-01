package bot

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rbhz/tg-dictionary/app/db"
	"github.com/rs/zerolog/log"
)

const (
	settingQuizType = "quiz_type"
)

// ListSettingsHandler handles /settings command
type ListSettingsHandler struct{}

func (h ListSettingsHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "settings"
}

func (h ListSettingsHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h ListSettingsHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	msg := tgbotapi.NewMessage(u.Message.From.ID, "Choose what do you want to change:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Quiz type", fmt.Sprintf("%v|%v", callbackIdSettings, settingQuizType)),
		),
	)
	b.Send(msg)
}

// SendQuizTypesHandler sends available quiz types
type SendQuizTypesHandler struct{}

func (h SendQuizTypesHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil &&
		u.CallbackQuery.Data == fmt.Sprintf("%v|%v", callbackIdSettings, settingQuizType)
}

func (h SendQuizTypesHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h SendQuizTypesHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	user, ok := ctx.Value(ctxUserKey).(db.User)
	if !ok {
		log.Error().Msg("invalid user in context")
		return
	}
	quizType := db.QuizTypeDefault
	if user.Config.QuizType != nil {
		quizType = *user.Config.QuizType
	}
	var quizTypeText string
	switch quizType {
	case db.QuizTypeTranslations:
		quizTypeText = "Translations"
	case db.QuizTypeReverseTranslations:
		quizTypeText = "Reverse translations"
	case db.QuizTypeMeanings:
		quizTypeText = "Meanings"
	}
	msg := tgbotapi.NewMessage(u.CallbackQuery.From.ID, fmt.Sprintf("Current type: %v\nPick quiz type:", quizTypeText))
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Translations", fmt.Sprintf("%v|%v|%v", callbackIdSettings, settingQuizType, db.QuizTypeTranslations),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Reverse translations", fmt.Sprintf("%v|%v|%v", callbackIdSettings, settingQuizType, db.QuizTypeReverseTranslations),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Meanings", fmt.Sprintf("%v|%v|%v", callbackIdSettings, settingQuizType, db.QuizTypeMeanings),
			),
		),
	)
	b.Send(msg)

}

// Set QuizTypeHandler saves quiz type to user config
type SetQuizTypesHandler struct{}

func (h SetQuizTypesHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil &&
		strings.HasPrefix(u.CallbackQuery.Data, fmt.Sprintf("%v|%v|", callbackIdSettings, settingQuizType))
}

func (h SetQuizTypesHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h SetQuizTypesHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	user, ok := ctx.Value(ctxUserKey).(db.User)
	if !ok {
		log.Error().Msg("invalid user in context")
		return
	}
	quizType := strings.Split(u.CallbackQuery.Data, "|")[2]
	user.Config.QuizType = &quizType
	switch quizType {
	case db.QuizTypeTranslations:
	case db.QuizTypeMeanings:
	case db.QuizTypeReverseTranslations:
	default:
		log.Error().Str("type", quizType).Msg("invalid quiz type")
		b.SendCallback(tgbotapi.NewCallback(u.CallbackQuery.ID, "Unknown type"))
	}
	b.DB().SaveUser(user)
	b.SendCallback(tgbotapi.NewCallback(u.CallbackQuery.ID, "Quiz type set"))
}
