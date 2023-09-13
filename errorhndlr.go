package http

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"runtime/debug"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/gofiber/fiber/v2"
	"github.com/gomig/logger"
	"github.com/gomig/utils"
	"github.com/inhies/go-bytesize"
)

// ErrorCallback a callback type for generate error response
type ErrorCallback func(*fiber.Ctx, error) error

// ErrorLogger handle errors and log into logger
//
// Enter only codes to log only codes included
func ErrorLogger(logger logger.Logger, formatter logger.TimeFormatter, callback ErrorCallback, onlyCodes ...int) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := 500
		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		detectMime := func(file *multipart.FileHeader) string {
			f, err := file.Open()
			if err == nil {
				defer f.Close()
				if mime, err := mimetype.DetectReader(f); err == nil && mime != nil {
					return mime.String()
				}
			}

			return "?"
		}

		// Log
		if logger != nil && (len(onlyCodes) == 0 || utils.Contains[int](onlyCodes, code)) {
			logger.Divider("=", 100, c.IP())
			logger.Error().Tags(fmt.Sprintf("%d", code)).Print(err.Error())
			logger.Raw("\n")
			logger.Divider("-", 100, "Stacktrace:")
			logger.Raw(string(debug.Stack()))
			logger.Raw("\n")
			logger.Divider("-", 100, "Request Header:")
			logger.Raw(c.Request().Header.String())
			logger.Raw("\n")
			logger.Divider("-", 100, "Request Body:")
			if form, err := c.MultipartForm(); form != nil && err == nil {
				values := make(map[string]string)
				for k, v := range form.Value {
					values[k] = utils.ConcatStr(", ", v...)
				}
				for k, files := range form.File {
					_files := make([]string, 0)
					for _, file := range files {
						bSize := bytesize.New(float64(file.Size))
						_files = append(_files, fmt.Sprintf("%s [%s] (%s)", file.Filename, bSize, detectMime(file)))
					}
					values[k] = utils.ConcatStr(", ", _files...)
				}
				_bytes, _ := json.MarshalIndent(values, "", "    ")
				logger.Raw(string(_bytes))
				form.RemoveAll()
			} else {
				logger.Raw(string(c.Request().Body()))
			}
			logger.Raw("\n")
			logger.Divider("=", 100, formatter(time.Now().UTC(), "2006-01-02 15:04:05"))
			logger.Raw("\n\n")
		}

		// Return response
		if callback == nil {
			return c.SendStatus(code)
		} else {
			return callback(c, err)
		}
	}
}
