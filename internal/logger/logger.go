package logger

import (
	"os"
	"time"

	"umineko_city_of_books/internal/config"

	"github.com/rs/zerolog"
)

var Log zerolog.Logger

func Init(level string) {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(parsed)

	Log = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime},
	).With().Timestamp().Logger()
}

type SettingsListener struct{}

func NewSettingsListener() *SettingsListener {
	return &SettingsListener{}
}

func (l *SettingsListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key != config.SettingLogLevel.Key {
		return
	}

	level, err := zerolog.ParseLevel(value)
	if err != nil {
		return
	}

	zerolog.SetGlobalLevel(level)
	Log.Info().Str("level", value).Msg("log level changed")
}
