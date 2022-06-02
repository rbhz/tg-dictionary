package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/rbhz/tg-dictionary/app/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

const quizChoiceLanguage = "ru"
const quizChoicesCount = 4
const quizMessageTemplate = `
<i>Word</i>: <b>{{ .quiz.DisplayWord }}</b>
<i>Choices</i>:
{{- range $choiceIdx, $choice := .quiz.Choices }}

<b>{{- if $.quiz.Result }}{{- if eq $.quiz.Result.Choice $choiceIdx }}☑️ {{- end }}{{- if $choice.Correct }}✅ {{- end }}{{- end }}{{inc $choiceIdx }}</b>: {{ $choice.Text }}
{{- end }}
`

// GetQuizMessageText returns text for quiz message
func GetQuizMessageText(quiz db.Quiz) (string, error) {
	funcMap := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}

	tmpl, err := template.New("template").Funcs(funcMap).Parse(quizMessageTemplate)
	if err != nil {
		return "", fmt.Errorf("parsing template: %w", err)
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, map[string]interface{}{"quiz": quiz}); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}
	return buf.String(), nil
}

// ErrNotEnoughWords is returned when there are not enough words in dictionary
var ErrNotEnoughWords = errors.New("not enough words")

// QuizHandler handles quiz command
type QuizHandler struct {
	neverPassthorugh
}

// Match returns true if update is /quiz command
func (h QuizHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "quiz"
}

// Handle generates new quiz and sends it to user
func (h QuizHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	user, ok := ctx.Value(ctxUserKey).(db.User)
	if !ok {
		log.Error().Msg("invalid user in context")
		return
	}
	quizType := db.QuizTypeDefault
	if user.Config.QuizType != nil {
		quizType = *user.Config.QuizType
	}
	dictionary, err := b.DB().GetUserDictionary(db.UserID(u.Message.From.ID))
	if err != nil {
		log.Error().Err(err).Int64("user", u.Message.From.ID).Msg("failed to get user dictionary")
	}
	if len(dictionary) == 0 {
		_, _ = b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "You don't have any words in your dictionary"))
		return
	}
	quizWord, err := h.getRandomWord(dictionary)
	if err != nil {
		log.Error().Err(err).Str("word", quizWord.Word).Msg("failed to get random word")
		return
	}
	choices, err := h.getChoices(quizWord, quizType, dictionary, quizChoicesCount)
	if err != nil {
		if errors.Is(err, ErrNotEnoughWords) {
			_, _ = b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Add more words to your dictionary"))
			return
		}
	}

	displayWord := quizWord.Word
	if quizType == db.QuizTypeReverseTranslations {
		for _, tr := range dictionary[quizWord].Translations {
			if tr.Language == quizChoiceLanguage {
				displayWord = tr.Text
				break
			}
		}
	}
	quiz := db.NewQuiz(db.UserID(u.Message.From.ID), quizWord.Word, displayWord, quizChoiceLanguage, choices, quizType)
	if err := b.DB().SaveQuiz(quiz); err != nil {
		log.Error().Err(err).Int64("user", u.Message.From.ID).Msg("failed to save quiz")
		return
	}
	text, err := GetQuizMessageText(quiz)
	if err != nil {
		log.Error().Err(err).Str("quiz", quiz.ID).Msg("failed to get text for message")
		return
	}
	message := tgbotapi.NewMessage(u.Message.Chat.ID, text)
	message.ParseMode = "html"
	message.ReplyMarkup = h.getMessageKeyboard(quiz)
	_, _ = b.Send(message)
}

// getRandomWord returns random word from dictionary based on last quiz time
func (h QuizHandler) getRandomWord(dict map[db.UserDictionaryItem]db.DictionaryItem) (db.UserDictionaryItem, error) {
	if len(dict) == 0 {
		return db.UserDictionaryItem{}, errors.New("empty dictionary")
	}
	var minTime, maxTime *time.Time
	items := make([]db.UserDictionaryItem, 0, len(dict))

	for item, word := range dict {
		var hasTranslation bool
		for _, t := range word.Translations {
			if t.Language == quizChoiceLanguage {
				hasTranslation = true
				break
			}
		}
		if !hasTranslation {
			continue
		}
		if len(dict[item].Translations) == 0 {
			continue
		}
		if item.LastQuiz != nil {
			if minTime == nil || item.LastQuiz.Before(*minTime) {
				minTime = item.LastQuiz
			}
			if maxTime == nil || item.LastQuiz.After(*maxTime) {
				maxTime = item.LastQuiz
			}
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		return db.UserDictionaryItem{}, ErrNotEnoughWords
	}

	if minTime == nil || maxTime == nil {
		return items[rand.Intn(len(items))], nil
	}
	weights := make(map[db.UserDictionaryItem]int, len(dict))
	timeRange := maxTime.Sub(*minTime).Seconds()
	var totalWeight uint
	for _, item := range items {
		weight := 100
		if item.LastQuiz != nil {
			if timeRange > 0 {
				weight = int(maxTime.Sub(*item.LastQuiz).Seconds() / timeRange * 100)
			} else {
				weight = 0
			}
		}
		if weight == 0 {
			weight = 1
		}
		totalWeight += uint(weight)
		weights[item] = weight
	}
	value := rand.Intn(int(totalWeight))
	result := db.UserDictionaryItem{}
	for _, item := range items {
		value -= weights[item]
		result = item
		if value <= 0 {
			break
		}
	}
	return result, nil
}

// getChoices returns random words from dictionary with same part of speech
func (h QuizHandler) getChoices(
	item db.UserDictionaryItem,
	qType string,
	dict map[db.UserDictionaryItem]db.DictionaryItem,
	count int,
) ([]db.QuizItem, error) {
	choices := make([]db.QuizItem, 0, len(dict))
	correctWord, ok := dict[item]
	if !ok {
		return choices, errors.New("item not in dictionary")
	}
	for _, word := range dict {
		if word.Word == item.Word {
			continue
		}
		// skip unsuitable words
		if qType == db.QuizTypeTranslations && len(word.Translations) == 0 {
			continue
		}
		if qType == db.QuizTypeMeanings && len(word.Meanings) == 0 {
			continue
		}

		choices = append(choices, db.QuizItem{
			Word:    word.Word,
			Text:    h.getWordChoiceText(word, qType),
			Correct: false})
	}
	if len(choices) < count {
		return choices, ErrNotEnoughWords
	}
	rand.Shuffle(len(choices), func(i, j int) { choices[i], choices[j] = choices[j], choices[i] })
	randomChoices := choices[:count-1]
	correctChoice := db.QuizItem{Word: correctWord.Word, Text: h.getWordChoiceText(correctWord, qType), Correct: true}
	randomChoices = append(randomChoices, correctChoice)

	sort.Slice(randomChoices, func(i, j int) bool { return randomChoices[i].Word < randomChoices[j].Word })
	return randomChoices, nil
}

// getWordChoiceText returns choice text based on quiz type
func (h QuizHandler) getWordChoiceText(word db.DictionaryItem, qType string) (text string) {
	switch qType {
	case db.QuizTypeTranslations:
		translations := make([]string, 0, len(word.Translations))
		for _, translation := range word.Translations {
			if translation.Language == quizChoiceLanguage {
				translations = append(translations, translation.Text)
			}
		}
		text = strings.Join(translations, ", ")
	case db.QuizTypeReverseTranslations:
		text = word.Word
	case db.QuizTypeMeanings:
		text = word.Meanings[0].Definition
	}
	return
}

// getMessageKeyboard returns keyboard with quiz choices
func (h QuizHandler) getMessageKeyboard(quiz db.Quiz) tgbotapi.InlineKeyboardMarkup {
	buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(quiz.Choices))
	for idx := range quiz.Choices {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d", idx+1),
			fmt.Sprintf("%v|%v|%d", callbackIDQuizReply, quiz.ID, idx)),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))
}

// QuizReplyHandler handles quiz reply callback
type QuizReplyHandler struct {
	neverPassthorugh
}

// Match returns true if update is quiz reply callback
func (h QuizReplyHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil && u.CallbackQuery.Data[:3] == fmt.Sprintf("%v|", callbackIDQuizReply)
}

// Handle checks if response is correct and saves it to quiz
func (h QuizReplyHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	quizID, choice, err := h.parseQuery(u)
	if err != nil {
		log.Error().Err(err).Str("query", u.CallbackQuery.Data).Msg("failed to parse callback query")
		return
	}
	quiz, err := b.DB().GetQuiz(quizID)
	if err != nil {
		log.Error().Err(err).Str("quiz", quizID).Msg("failed to get quiz")
	}
	if quiz.User != db.UserID(u.CallbackQuery.From.ID) {
		_, _ = b.SendCallback(tgbotapi.NewCallback(u.CallbackQuery.ID, "Unknown quiz"))
		return
	}
	if err := quiz.SetResult(choice, b.DB()); err != nil {
		log.Error().Err(err).Str("quiz", quiz.ID).Int("choice", choice).Msg("failed to set quiz result")
		response := tgbotapi.NewCallback(u.CallbackQuery.ID, "Error happened")
		_, _ = b.SendCallback(response)
		return
	}
	if quiz.Result.Correct {
		_, _ = b.Send(tgbotapi.NewMessage(u.CallbackQuery.From.ID, "Correct!"))
	} else {
		_, _ = b.Send(tgbotapi.NewMessage(u.CallbackQuery.From.ID, "Wrong!"))
		h.sendItemMessage(quiz.Word, u.CallbackQuery.From.ID, b)
	}
	quizText, err := GetQuizMessageText(quiz)
	if err != nil {
		log.Error().Err(err).Str("quiz", quiz.ID).Msg("failed to get quiz message text")
	}
	edit := tgbotapi.NewEditMessageText(
		u.CallbackQuery.From.ID,
		u.CallbackQuery.Message.MessageID,
		quizText)
	edit.ReplyMarkup = nil
	edit.ParseMode = "html"
	_, _ = b.Send(edit)
}

func (h QuizReplyHandler) sendItemMessage(word string, user int64, b Bot) {
	item, err := b.DB().Get(word)
	if err != nil {
		log.Error().Err(err).Str("word", word).Msg("failed to get item")
	}
	text := GetItemMessageText(item)
	msg := tgbotapi.NewMessage(user, text)
	msg.ParseMode = "html"
	_, _ = b.Send(msg)
}

func (h QuizReplyHandler) parseQuery(u tgbotapi.Update) (ID string, choice int, err error) {
	parts := strings.Split(u.CallbackQuery.Data, "|")
	if len(parts) != 3 {
		return "", 0, errors.New("invalid callback query data")
	}
	ID = parts[1]
	choice, err = strconv.Atoi(parts[2])
	if err != nil {
		return "", 0, fmt.Errorf("parsing choice: %w", err)
	}
	return ID, choice, nil
}
