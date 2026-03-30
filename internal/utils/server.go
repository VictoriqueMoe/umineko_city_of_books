package utils

import (
	"os"
	"os/signal"
	"syscall"

	"umineko_city_of_books/internal/logger"

	"github.com/gofiber/fiber/v3"
)

func StartServerWithGracefulShutdown(app *fiber.App, addr string) {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		if err := app.Shutdown(); err != nil {
			logger.Log.Error().Err(err).Msg("server shutdown error")
		}

		close(idleConnsClosed)
	}()

	if err := app.Listen(addr); err != nil {
		logger.Log.Error().Err(err).Msg("server error")
	}

	<-idleConnsClosed
}
