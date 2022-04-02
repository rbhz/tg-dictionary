package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"tg-dictionary/app/clients/dictionaryapi"
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
	if dbItem == nil {
		client := dictionaryapi.NewDictionaryAPIClient(nil)
		dictionary, err := client.Get(word)
		if err != nil {
			if errors.Is(err, dictionaryapi.ErrNotFound) {
				if _, err := b.api.Send(
					tgbotapi.NewMessage(u.Message.From.ID, "Unknown word"),
				); err != nil {
					log.Error().Err(err).Msg("failed to send new word reply")
				}

			}
			log.Error().Err(err).Str("word", word).Msg("failed to get dictionary info")
			return
		}
		item = db.NewDictionaryItem(word, dictionary)
		if err := b.db.Save(item); err != nil {
			log.Error().Err(err).Msg("failed to save dictionary item")
			return
		}
	} else {
		item = *dbItem
	}
	userID := db.UserID(u.Message.From.ID)
	b.db.SaveForUser(item, userID)
	fmt.Printf("%+v\n", item)

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
