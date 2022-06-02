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
type ListSettingsHandler struct {
	neverPassthorugh
}

// Match returns true if update is /settings command
func (h ListSettingsHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "settings"
}

// Handle sends settings list keyboard
func (h ListSettingsHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	msg := tgbotapi.NewMessage(u.Message.From.ID, "Choose what do you want to change:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Quiz type", fmt.Sprintf("%v|%v", callbackIDSettings, settingQuizType)),
		),
	)
	_, _ = b.Send(msg)
}

// SendQuizTypesHandler sends available quiz types
type SendQuizTypesHandler struct {
	neverPassthorugh
}

// Match returns true if update is quiz settings callback
func (h SendQuizTypesHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil &&
		u.CallbackQuery.Data == fmt.Sprintf("%v|%v", callbackIDSettings, settingQuizType)
}

// Handle sends settings quiz type lists keyboard
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
				"Translations", fmt.Sprintf("%v|%v|%v", callbackIDSettings, settingQuizType, db.QuizTypeTranslations),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Reverse translations", fmt.Sprintf("%v|%v|%v", callbackIDSettings, settingQuizType, db.QuizTypeReverseTranslations),
			),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				"Meanings", fmt.Sprintf("%v|%v|%v", callbackIDSettings, settingQuizType, db.QuizTypeMeanings),
			),
		),
	)
	_, _ = b.Send(msg)

}

// SetQuizTypesHandler saves quiz type to user config
type SetQuizTypesHandler struct {
	neverPassthorugh
}

// Match returns true if update is quiz settings callback with picked type
func (h SetQuizTypesHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil &&
		strings.HasPrefix(u.CallbackQuery.Data, fmt.Sprintf("%v|%v|", callbackIDSettings, settingQuizType))
}

// Handle saves quiz type to user config
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
		_, _ = b.SendCallback(tgbotapi.NewCallback(u.CallbackQuery.ID, "Unknown type"))
	}
	if err := b.DB().SaveUser(user); err != nil {
		log.Error().Err(err).Msg("failed to save user")
		return
	}
	_, _ = b.SendCallback(tgbotapi.NewCallback(u.CallbackQuery.ID, "Quiz type set"))
}
