package bot

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"strings"
	"tg-dictionary/app/clients/dictionaryapi"
	"tg-dictionary/app/clients/mymemory"
	"tg-dictionary/app/db"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/rs/zerolog/log"
)

func startHandler(ctx context.Context, b *TelegramBot, u tgbotapi.Update) {
	m := tgbotapi.NewMessage(u.Message.From.ID, "Hi! Just send me a word!")
	if _, err := b.api.Send(m); err != nil {
		log.Error().Err(err).Msg("failed to send start reply")
	}
}

func addToDictHandler(ctx context.Context, b *TelegramBot, u tgbotapi.Update) {
	word := strings.ToLower(u.Message.Text)
	if strings.Contains(word, " ") {
		if _, err := b.api.Send(
			tgbotapi.NewMessage(u.Message.From.ID, "Sorry only single words are supported"),
		); err != nil {
			log.Error().Err(err).Msg("failed to send new word reply")
		}
	}
	dbItem, err := b.db.Get(word)
	if err != nil {
		log.Error().Err(err).Msg("failed to get dictionary item")
		return
	}
	var item db.DictionaryItem
	if dbItem != nil {
		item = *dbItem
	} else {
		dictOut, dictErrChan := make(chan []dictionaryapi.WordResponse), make(chan error)
		translationOut, translationErrChar := make(chan mymemory.TranslationResponse), make(chan error)
		go func() {
			client := dictionaryapi.NewDictionaryAPIClient(ctx)
			dictionary, err := client.Get(word)
			dictOut <- dictionary
			dictErrChan <- err
		}()
		go func() {
			client := mymemory.NewMymemoryClient(ctx, nil)
			translation, err := client.Translate(word, "en", "ru")
			translationOut <- translation
			translationErrChar <- err
		}()
		tranlation, tranlationErr := <-translationOut, <-translationErrChar
		dictionary, dictErr := <-dictOut, <-dictErrChan
		if dictErr != nil {
			if errors.Is(dictErr, dictionaryapi.ErrNotFound) {
				if _, err := b.api.Send(
					tgbotapi.NewMessage(u.Message.From.ID, "Unknown word"),
				); err != nil {
					log.Error().Err(err).Msg("failed to send new word reply")
				}
				return
			}
			log.Error().Err(err).Str("word", word).Msg("failed to get dictionary info")
			return
		}
		tranlationPtr := &tranlation
		if tranlationErr != nil {
			if !errors.Is(tranlationErr, mymemory.ErrUnknown) {
				log.Error().Err(err).Str("word", word).Msg("failed to get translation info")
			}
			tranlationPtr = nil
		}
		item = db.NewDictionaryItem(word, dictionary, tranlationPtr)
		if err := b.db.Save(item); err != nil {
			log.Error().Err(err).Msg("failed to save dictionary item")
			return
		}
	}
	userID := db.UserID(u.Message.From.ID)
	b.db.SaveForUser(item, userID)

	tmpl, err := template.New("template").Parse(DictionaryItemTemplate)
	if err != nil {
		log.Error().Err(err).Msg("failed to parse item template")
		return
	}
	buf := &bytes.Buffer{}
	if err := tmpl.Execute(buf, map[string]interface{}{"Item": item}); err != nil {
		log.Error().Err(err).Msg("failed to format item template")
	}
	text := tgbotapi.NewMessage(u.Message.From.ID, buf.String())
	text.ParseMode = "html"
	if _, err := b.api.Send(text); err != nil {
		log.Error().Err(err).Msg("failed to send dictionary item reply")
	}
	if item.Phonetics.Audio != "" {
		audio := tgbotapi.NewAudio(u.Message.From.ID, tgbotapi.FileURL(item.Phonetics.Audio))
		if _, err := b.api.Send(audio); err != nil {
			log.Error().Err(err).Msg("failed to send dictionary item audio reply")
		}
	}
}
