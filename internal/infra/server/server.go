package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/platform/validator"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)
var Invoke = fx.Invoke(Start)

type Params struct {
	fx.In
	Lifecycle fx.Lifecycle
	Validator validator.Interface
	Env       *env.Env
}

func New(params Params) (*fiber.App, fiber.Router, error) {
	app := fiber.New(fiber.Config{
		StrictRouting:   true,
		CaseSensitive:   true,
		AppName:         "sublyra-api",
		StructValidator: params.Validator,
		ErrorHandler:    ErrorHandler,
	})

	registerMiddlewares(app, params.Env)

	return app, app.Group("/api/v1"), nil
}

func registerMiddlewares(app *fiber.App, env *env.Env) {
	app.Use(recover.New(recover.ConfigDefault))
	app.Use(requestid.New())
	app.Use(logger.New())
	// app.Use(cors.New(cors.Config{
	// 	AllowOrigins: []string{env.CorsAllowedOrigins},
	// 	AllowMethods: []string{fiber.MethodGet, fiber.MethodPost},
	// 	AllowHeaders: []string{fiber.HeaderOrigin, fiber.HeaderContentType, fiber.HeaderAccept},
	// }))
	app.Use(limiter.New(limiter.Config{
		Expiration:             30 * time.Second,
		SkipSuccessfulRequests: false,
		Max:                    3,
	}))
}

func ErrorHandler(c fiber.Ctx, err error) error {
	code := ports.CodeInternal

	status := fiber.StatusInternalServerError
	attr := []any{slog.String("type", "unknown")}

	var f *ports.Error
	var e *fiber.Error
	var message string

	switch {
	case errors.As(err, &f):
		status = f.Status
		code = f.Code
		message = string(f.Message)
		attr = []any{
			slog.String("type", "domain"),
			slog.String("code", string(f.Code)),
		}

	case errors.As(err, &e):
		status = e.Code
		message = e.Message
		attr = []any{
			slog.String("type", "fiber"),
			slog.Int("status", e.Code),
		}
	}

	slog.Error(
		fmt.Sprintf("[%s:%d] %s", code, status, message),
		slog.String("request_id", requestid.FromContext(c)),
		slog.String("method", c.Method()),
		slog.String("path", c.Path()),
		slog.Int("status", status),
		slog.Any("error", err),
		slog.Group("context", attr...),
	)

	return c.Status(status).JSON(fiber.Map{
		"code":    code,
		"message": message,
	})

}

func Start(lifecycle fx.Lifecycle, app *fiber.App, logger *slog.Logger, env *env.Env) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(env.ServerHost + ":" + env.ServerPort); err != nil {
					logger.Error("error starting the HTTP server", "error", err.Error())
				}
			}()

			return nil
		},
		OnStop: app.ShutdownWithContext,
	})
}
