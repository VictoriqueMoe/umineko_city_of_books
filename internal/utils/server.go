package utils

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
)

func StartServerWithGracefulShutdown(app *fiber.App, addr string) {
	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		if err := app.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}

		close(idleConnsClosed)
	}()

	if err := app.Listen(addr); err != nil {
		log.Printf("Server error: %v", err)
	}

	<-idleConnsClosed
}
