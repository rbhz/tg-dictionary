package bot

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"github.com/rbhz/tg-dictionary/app/clients/dictionaryapi"
	"github.com/rbhz/tg-dictionary/app/clients/ya_dictionary"
	"github.com/rbhz/tg-dictionary/app/db"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

const (
	fromLanguage = "en"
	toLanguage   = "ru"
)

const dictionaryItemTemplate = `<b>{{ .Item.Word }}</b>
{{- if .Item.Translations }}
<b>Translations</b>:
{{- range $t := .Item.Translations }} <code>{{ $t.Text }}</code>({{ $t.Language }}){{- end }}
___
{{- end }}
<b>Meanings:</b>
{{- range  $m := .Item.Meanings }}
<code>{{ $m.Definition }}</code> ({{ $m.PartOfSpeech }})
{{- range $e := $m.Examples }}
{{ $e }}
{{- end }}
___
{{- end }}
<u>Phonetics</u>: {{ .Item.Phonetics.Text }}
`

func GetItemMessageText(item db.DictionaryItem) string {
	tmpl, err := template.New("template").Parse(dictionaryItemTemplate)
	if err != nil {
		log.Error().Err(err).Str("word", item.Word).Msg("failed to parse item template")
		return ""
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, map[string]interface{}{"Item": item}); err != nil {
		log.Error().Err(err).Str("word", item.Word).Msg("failed to format item template")
	}
	return buf.String()
}

type WordHandler struct {
	tranlationsToken string
}

func (h WordHandler) Passthrough(u tgbotapi.Update) bool {
	return false
}

func (h WordHandler) Match(u tgbotapi.Update) bool {
	return u.Message != nil && u.Message.Text != "" && !u.Message.IsCommand()
}

func (h WordHandler) Handle(ctx context.Context, b Bot, u tgbotapi.Update) {
	word := strings.ToLower(u.Message.Text)
	if strings.Contains(word, " ") {
		b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Sorry only single words are supported"))
		return
	}
	userID := db.UserID(u.Message.From.ID)
	item, err := h.getItemData(ctx, word, b.DB())
	if err != nil {
		log.Error().Err(err).Str("word", word).Msg("failed to get word data")
		return
	}
	if item == nil {
		b.Send(tgbotapi.NewMessage(u.Message.From.ID, "Sorry, I don't know this word"))
		return
	}

	if _, err := b.DB().GetUserItem(userID, item.Word); err != nil && errors.Is(err, db.ErrNotFound) {
		b.DB().SaveUserItem(db.UserDictionaryItem{
			User:    userID,
			Word:    item.Word,
			Created: time.Now(),
		})
	}

	text := tgbotapi.NewMessage(u.Message.From.ID, GetItemMessageText(*item))
	text.ParseMode = "html"
	if _, err := b.Send(text); err == nil && item.Phonetics.Audio != "" {
		audio := tgbotapi.NewAudio(u.Message.From.ID, tgbotapi.FileURL(item.Phonetics.Audio))
		b.Send(audio)
	}
}

func (h WordHandler) getItemData(ctx context.Context, word string, storage db.Storage) (*db.DictionaryItem, error) {
	dbItem, err := storage.Get(word)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		return nil, fmt.Errorf("fetch from db: %w", err)
	}

	var item db.DictionaryItem
	if err == nil {
		return &dbItem, nil
	} else {
		dictOut, dictErrChan := make(chan []dictionaryapi.WordResponse), make(chan error)
		translationOut, translationErrChar := make(chan ya_dictionary.TranslationResponse), make(chan error)
		go func() {
			client := dictionaryapi.NewDictionaryAPIClient(ctx)
			dictionary, err := client.Get(word)
			dictOut <- dictionary
			dictErrChan <- err
		}()
		go func() {
			client := ya_dictionary.NewYaDictionaryClient(ctx, h.tranlationsToken)
			translation, err := client.Translate(word, fromLanguage, toLanguage)
			translationOut <- translation
			translationErrChar <- err
		}()
		translation, translationErr := <-translationOut, <-translationErrChar
		dictionary, dictErr := <-dictOut, <-dictErrChan
		if dictErr != nil {
			if errors.Is(dictErr, dictionaryapi.ErrNotFound) {
				return nil, nil
			}
			return nil, fmt.Errorf("get dictionary info: %w", dictErr)
		}
		translations := make(map[string]ya_dictionary.TranslationResponse, 1)
		if translationErr != nil {
			if !errors.Is(translationErr, ya_dictionary.ErrUnknown) {
				log.Error().Err(err).Str("word", word).Msg("failed to get translation info")
			}
		} else {
			translations[toLanguage] = translation
		}
		item = db.NewDictionaryItem(word, dictionary, translations)
		if err := storage.Save(item); err != nil {
			return nil, fmt.Errorf("save to db: %w", err)
		}
		return &item, nil
	}
}

func NewWordHandler(tranlationsToken string) WordHandler {
	return WordHandler{tranlationsToken: tranlationsToken}
}
