package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	bea "github.com/rickferrdev/bea-go"
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
	Log       *bea.Logger
}

func New(params Params) (*fiber.App, fiber.Router, error) {
	app := fiber.New(fiber.Config{
		StrictRouting:   true,
		CaseSensitive:   true,
		AppName:         "sublyra-api",
		StructValidator: params.Validator,
		ErrorHandler:    NewErrorHandler(params.Log),
	})

	registerMiddlewares(app, params.Env, params.Log)

	return app, app.Group("/api/v1"), nil
}

func registerMiddlewares(app *fiber.App, env *env.Env, logger *bea.Logger) {
	app.Use(recover.New(recover.ConfigDefault))
	app.Use(requestid.New())
	app.Use(ResponseLogger(logger))
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

func ResponseLogger(logger *bea.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)

		fields := []bea.Field{
			bea.String("request_id", requestid.FromContext(c)),
			bea.String("method", c.Method()),
			bea.String("path", c.Path()),
			bea.Int("status", c.Response().StatusCode()),
			bea.String("latency", latency.String()),
		}

		if err != nil {
			return err
		}

		if c.Response().StatusCode() >= fiber.StatusInternalServerError {
			_ = logger.ErrorContext(c.Context(), "http response completed", fields...)
			return nil
		}

		_ = logger.InfoContext(c.Context(), "http response completed", fields...)
		return nil
	}
}

func NewErrorHandler(logger *bea.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		code := ports.CodeInternal

		status := fiber.StatusInternalServerError
		attr := []bea.Field{bea.String("type", "unknown")}

		var f *ports.Error
		var e *fiber.Error
		var message string

		switch {
		case errors.As(err, &f):
			status = f.Status
			code = f.Code
			message = string(f.Message)
			attr = []bea.Field{
				bea.String("type", "domain"),
				bea.String("code", string(f.Code)),
			}

		case errors.As(err, &e):
			status = e.Code
			message = e.Message
			attr = []bea.Field{
				bea.String("type", "fiber"),
				bea.Int("status", e.Code),
			}
		}

		fields := []bea.Field{
			bea.String("request_id", requestid.FromContext(c)),
			bea.String("method", c.Method()),
			bea.String("path", c.Path()),
			bea.Int("status", status),
			bea.Any("error", err),
		}
		fields = append(fields, attr...)

		_ = logger.ErrorContext(c.Context(), fmt.Sprintf("[%s:%d] %s", code, status, message), fields...)

		return c.Status(status).JSON(fiber.Map{
			"code":    code,
			"message": message,
		})
	}
}

func Start(lifecycle fx.Lifecycle, app *fiber.App, logger *bea.Logger, env *env.Env) {
	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				if err := app.Listen(env.ServerHost + ":" + env.ServerPort); err != nil {
					_ = logger.Error("error starting the HTTP server", bea.String("error", err.Error()))
				}
			}()

			return nil
		},
		OnStop: app.ShutdownWithContext,
	})
}
