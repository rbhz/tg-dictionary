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
	"github.com/rbhz/tg-dictionary/app/db"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

const quizReplyCallbackID = "qr"
const quizChoiceLanguage = "ru"
const quizChoicesCount = 4
const quizMessageTemplate = `
<i>Word</i>: <b>{{ .quiz.Word }}</b>
<i>Choices</i>:
{{- range $choiceIdx, $choice := .quiz.Choices }}

<b>{{- if $.quiz.Result }}{{- if eq $.quiz.Result.Choice $choiceIdx }}☑️ {{- end }}{{- if $choice.Correct }}✅ {{- end }}{{- end }}{{inc $choiceIdx }}</b>:
{{- range $i, $w := $choice.Translations }}{{ if $i }},{{- end }} {{ $w }}{{- end }}
{{- end }}
`

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

var ErrNotEnoughWords = errors.New("not enough words")

type QuizHandler struct{}

func (h QuizHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Command() == "quiz"
}

func (h QuizHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h QuizHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	dictionary, err := b.DB().GetUserDictionary(db.UserID(u.Message.From.ID))
	if err != nil {
		log.Error().Err(err).Int64("user", u.Message.From.ID).Msg("failed to get user dictionary")
	}
	if len(dictionary) == 0 {
		b.Send(tgbotapi.NewMessage(u.Message.Chat.ID, "You don't have any words in your dictionary"))
		return
	}
	quizWord, err := h.getRandomWord(dictionary)
	if err != nil {
		log.Error().Err(err).Str("word", quizWord.Word).Msg("failed to get random word")
		return
	}
	choices, err := h.getChoices(quizWord, dictionary, quizChoicesCount)
	if err != nil {
		if errors.Is(err, ErrNotEnoughWords) {
			b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Add more words to your dictionary"))
			return
		}
	}
	quiz := db.NewQuiz(db.UserID(u.Message.From.ID), quizWord.Word, quizChoiceLanguage, choices)
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
	b.Send(message)
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
		if len(word.Translations) == 0 {
			continue
		}
		translations := make([]string, 0, len(word.Translations))
		for _, translation := range word.Translations {
			if translation.Language == quizChoiceLanguage {
				translations = append(translations, translation.Text)
			}
		}
		if len(translations) > 0 {
			choices = append(choices, db.QuizItem{Word: word.Word, Translations: translations, Correct: false})
		}
	}
	if len(choices) < count {
		return choices, ErrNotEnoughWords
	}
	rand.Shuffle(len(choices), func(i, j int) { choices[i], choices[j] = choices[j], choices[i] })
	randomChoices := choices[:count-1]
	correctChoice := db.QuizItem{Word: correctWord.Word, Translations: make([]string, 0, len(correctWord.Translations)), Correct: true}
	for _, translation := range correctWord.Translations {
		if translation.Language == quizChoiceLanguage {
			correctChoice.Translations = append(correctChoice.Translations, translation.Text)
		}
	}
	randomChoices = append(randomChoices, correctChoice)

	sort.Slice(randomChoices, func(i, j int) bool { return randomChoices[i].Word < randomChoices[j].Word })
	return randomChoices, nil
}

func (h QuizHandler) getMessageKeyboard(quiz db.Quiz) tgbotapi.InlineKeyboardMarkup {
	buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(quiz.Choices))
	for idx := range quiz.Choices {
		buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d", idx+1),
			fmt.Sprintf("%v|%v|%d", quizReplyCallbackID, quiz.ID, idx)),
		)
	}
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(buttons...))
}

// QuizReplyCallbackHandler handles quiz reply callback
type QuizReplyHandler struct{}

func (h QuizReplyHandler) Match(u tgbotapi.Update) bool {
	return u.CallbackQuery != nil && u.CallbackQuery.Data[:3] == fmt.Sprintf("%v|", quizReplyCallbackID)
}

func (h QuizReplyHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

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
		b.Send(tgbotapi.NewCallback(u.CallbackQuery.ID, "Unknown quiz"))
		return
	}
	if err := quiz.SetResult(choice, b.DB()); err != nil {
		log.Error().Err(err).Str("quiz", quiz.ID).Int("choice", choice).Msg("failed to set quiz result")
		response := tgbotapi.NewCallback(u.CallbackQuery.ID, "Error happend")
		b.Send(response)
		return
	}
	if quiz.Result.Correct {
		b.Send(tgbotapi.NewMessage(u.CallbackQuery.From.ID, "Correct!"))
	} else {
		b.Send(tgbotapi.NewMessage(u.CallbackQuery.From.ID, "Wrong!"))
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
	b.Send(edit)
}

func (h QuizReplyHandler) sendItemMessage(word string, user int64, b Bot) {
	item, err := b.DB().Get(word)
	if err != nil {
		log.Error().Err(err).Str("word", word).Msg("failed to get item")
	}
	text := GetItemMessageText(item)
	msg := tgbotapi.NewMessage(user, text)
	msg.ParseMode = "html"
	b.Send(msg)
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
