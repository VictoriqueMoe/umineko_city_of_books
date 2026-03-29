package logger

import (
	"os"
	"time"

	"umineko_city_of_books/internal/config"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init() {
	level, err := zerolog.ParseLevel(config.Cfg.LogLevel)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	Log = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime},
	).With().Timestamp().Logger()
}
