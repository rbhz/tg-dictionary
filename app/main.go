package main

import (
	"os"

	"github.com/rbhz/tg-dictionary/app/api"
	"github.com/rbhz/tg-dictionary/app/bot"
	"github.com/rbhz/tg-dictionary/app/db"

	"github.com/jessevdk/go-flags"
	log "github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

type Opts struct {
	BotToken              string `long:"bot-token" env:"BOT_TOKEN" required:"true" description:"Telegram bot token"`
	BoltDB                string `long:"boltdb" env:"BOLTDB" default:"./dict.data" description:"Path to BoltDB"`
	RedisURL              string `long:"redis" env:"REDIS_URL" description:"Redis database URL"`
	YandexDictionaryToken string `long:"yadict-token" env:"YANDEX_DICTIONARY_TOKEN" required:"true" description:"Yandex Dictionary token"`
	JWTSecret             string `long:"jwt" env:"JWT_SECRET" required:"true" description:"JWT secret"`
	Port                  int    `long:"port" env:"PORT" default:"8080" description:"Port to listen on"`
}

func main() {
	var opts Opts
	_, err := flags.ParseArgs(&opts, os.Args)
	if err != nil {
		return
	}

	storage, closeStorage := getStorage(opts)
	defer closeStorage()

	// Start API
	go func() {
		api := api.NewServer(storage, opts.BotToken, opts.JWTSecret)
		if err := api.Run(opts.Port); err != nil {
			log.Fatal().Err(err).Msg("failed to run API server")
		}
	}()

	// initialize Telegram bot
	b, err := bot.NewTelegramBot(opts.BotToken, storage, []bot.Handler{
		bot.StartHandler{},
		// Settings
		bot.ListSettingsHandler{},
		bot.SendQuizTypesHandler{},
		bot.SetQuizTypesHandler{},
		// Quizzes
		bot.QuizHandler{},
		bot.QuizReplyHandler{},
		// Dictionary
		bot.NewWordHandler(opts.YandexDictionaryToken),
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize telegram bot")
	}
	b.Start()

}

func getStorage(opts Opts) (db.Storage, func()) {
	if opts.RedisURL != "" {
		redisStorage, err := db.NewRedisStorage(opts.RedisURL)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create redis client")
		}
		return redisStorage, func() {}

	} else {
		boltDB, err := bolt.Open(opts.BoltDB, 0600, nil)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create boltDB database")
		}
		boltStorage, err := db.NewBoltStorage(boltDB)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to bolt storage")
		}
		return boltStorage, func() {
			err := boltDB.Close()
			if err != nil {
				log.Error().Err(err).Msg("failed to close boltDB database")
			}
		}
	}
}
