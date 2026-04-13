package utils

import "github.com/gofiber/fiber/v3"

// BindJSON decodes the request body into T. On failure it writes a 400
// response and returns the zero value + false; callers should then `return nil`.
func BindJSON[T any](ctx fiber.Ctx) (T, bool) {
	var req T
	if err := ctx.Bind().JSON(&req); err != nil {
		_ = BadRequest(ctx, "invalid request body")
		return req, false
	}
	return req, true
}
