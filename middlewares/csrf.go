package middlewares

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gomig/http/session"
	"github.com/gomig/utils"
	"github.com/google/uuid"
)

// CSRFMiddleware protection middleware
func CSRFMiddleware(session session.Session) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		if session == nil {
			return utils.TaggedError([]string{"CSRFMiddleware"}, "session driver is nil")
		}

		if !session.Exists("csrf_token") {
			session.Set("csrf_token", uuid.New().String())
		}

		return ctx.Next()
	}
}
