package utils

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"umineko_city_of_books/internal/logger"

	"github.com/gofiber/fiber/v3"
)

const (
	httpShutdownTimeout = 8 * time.Second
)

func StartServerWithGracefulShutdown(app *fiber.App, addr string) error {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		if err := app.ShutdownWithTimeout(httpShutdownTimeout); err != nil {
			logger.Log.Error().Err(err).Msg("server shutdown error")
		}

		close(idleConnsClosed)
	}()

	if err := app.Listen(addr); err != nil {
		return err
	}

	<-idleConnsClosed

	return nil
}
