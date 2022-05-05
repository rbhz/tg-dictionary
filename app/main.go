package main

import (
	"os"
	"github.com/rbhz/tg-dictionary/app/bot"
	"github.com/rbhz/tg-dictionary/app/db"

	"github.com/jessevdk/go-flags"
	log "github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

type Opts struct {
	BotToken              string `long:"bot-token" env:"BOT_TOKEN" required:"true" description:"Telegram bot token"`
	BoltDB                string `long:"boltdb" env:"BOLTDB" default:"./dict.data" description:"Path to BoltDB"`
	YandexDictionaryToken string `long:"yadict-token" env:"YANDEX_DICTIONARY_TOKEN" required:"true" description:"Yandex Dictionary token"`
}

func main() {
	var opts Opts
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}
	// initialize DB
	boltDB, err := bolt.Open(opts.BoltDB, 0600, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create boltDB database")
	}
	defer boltDB.Close()
	storage, err := db.NewBoltStorage(boltDB)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to bolt storage")
	}

	// initialize Telegram bot
	b, err := bot.NewTelegramBot(opts.BotToken, storage, []bot.Handler{
		bot.StartHandler{},
		bot.QuizHandler{},
		bot.QuizReplyHandler{},
		bot.NewWordHandler(opts.YandexDictionaryToken),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize telegram bot")
	}
	b.Start()

}
