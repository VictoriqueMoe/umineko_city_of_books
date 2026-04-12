package logger

import (
	"os"
	"sync"
	"time"

	"umineko_city_of_books/internal/config"

	"github.com/getsentry/sentry-go"
	sentryzerolog "github.com/getsentry/sentry-go/zerolog"
	"github.com/rs/zerolog"
)

var (
	Log           zerolog.Logger
	console       zerolog.ConsoleWriter
	sentryWriter  *sentryzerolog.Writer
	shipper       *glitchtipShipper
	currentDSN    string
	reconfigureMu sync.Mutex
)

func Init(level string) {
	parsed, err := zerolog.ParseLevel(level)
	if err != nil {
		parsed = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(parsed)

	console = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.DateTime}
	applySentry(config.SettingSentryDSN.Default)
}

func applySentry(dsn string) {
	reconfigureMu.Lock()
	defer reconfigureMu.Unlock()

	if dsn == currentDSN && (sentryWriter != nil || dsn == "") {
		return
	}
	currentDSN = dsn

	if sentryWriter != nil {
		sentryWriter.Close()
		sentryWriter = nil
	}
	if shipper != nil {
		shipper.stop()
		shipper = nil
	}

	if dsn == "" {
		Log = zerolog.New(console).With().Timestamp().Logger()
		return
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      "production",
		EnableTracing:    true,
		TracesSampleRate: 0.15,
	}); err != nil {
		Log = zerolog.New(console).With().Timestamp().Logger()
		Log.Warn().Err(err).Msg("failed to initialise sentry client")
		return
	}

	sw, err := sentryzerolog.NewWithHub(sentry.CurrentHub(), sentryzerolog.Options{
		Levels:          []zerolog.Level{zerolog.ErrorLevel, zerolog.FatalLevel},
		WithBreadcrumbs: true,
		FlushTimeout:    2 * time.Second,
	})
	if err != nil {
		Log = zerolog.New(console).With().Timestamp().Logger()
		Log.Warn().Err(err).Msg("failed to initialise sentry writer, falling back to console-only")
		return
	}

	sentryWriter = sw

	newShipper, shipperErr := newGlitchtipShipper(dsn)
	if shipperErr != nil {
		Log = zerolog.New(zerolog.MultiLevelWriter(console, sw)).With().Timestamp().Logger()
		Log.Warn().Err(shipperErr).Msg("failed to create glitchtip log shipper, logs tab will be empty")
		return
	}

	newShipper.start()
	shipper = newShipper

	Log = zerolog.New(zerolog.MultiLevelWriter(console, sw, newShipper)).With().Timestamp().Logger()
	Log.Info().Msg("sentry/glitchtip error reporting enabled")
}

func Shutdown() {
	if shipper != nil {
		shipper.stop()
	}
	if sentryWriter != nil {
		sentryWriter.Close()
	}
	sentry.Flush(2 * time.Second)
}

type SettingsListener struct{}

func NewSettingsListener() *SettingsListener {
	return &SettingsListener{}
}

func (l *SettingsListener) OnSettingChanged(key config.SiteSettingKey, value string) {
	if key == config.SettingLogLevel.Key {
		level, err := zerolog.ParseLevel(value)
		if err != nil {
			return
		}
		zerolog.SetGlobalLevel(level)
		Log.Info().Str("level", value).Msg("log level changed")
		return
	}

	if key == config.SettingSentryDSN.Key {
		applySentry(value)
		return
	}
}
