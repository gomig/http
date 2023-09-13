package middlewares

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gomig/logger"
)

// AccessLogger middleware
func AccessLogger(logger logger.Logger) fiber.Handler {
	return func(ctx *fiber.Ctx) error {
		defer func(start time.Time) {
			stop := time.Now()
			latecy := stop.Sub(start).String()
			logger.
				Log().
				Type(ctx.Method()).
				Tags(fmt.Sprint(ctx.Response().StatusCode())).
				Tags(ctx.IP()).
				Tags(latecy).
				Print(ctx.Path())
		}(time.Now())
		return ctx.Next()
	}
}
