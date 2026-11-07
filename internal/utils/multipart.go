package utils

import (
	"mime"

	"github.com/gofiber/fiber/v3"
)

func IsMultipart(ctx fiber.Ctx) bool {
	mediaType, _, err := mime.ParseMediaType(ctx.Get(fiber.HeaderContentType))
	if err != nil {
		return false
	}

	return mediaType == fiber.MIMEMultipartForm
}
