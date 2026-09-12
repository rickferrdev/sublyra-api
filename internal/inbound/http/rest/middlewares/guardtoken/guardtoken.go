package guardtoken

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/constants"
	"github.com/rickferrdev/sublyra-api/internal/platform/jwttoken"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

type Middleware struct {
	jwttoken jwttoken.Interface
}

type FxParams struct {
	fx.In
	JwtToken jwttoken.Interface
}

func New(params FxParams) (*Middleware, error) {
	middleware := Middleware{
		jwttoken: params.JwtToken,
	}

	return &middleware, nil
}

func (handler *Middleware) GuardToken(c fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return ports.Unauthorized(nil)
	}

	claims, err := handler.jwttoken.ValidateToken(token)
	if err != nil {
		return ports.Unauthorized(nil)
	}

	ctx := context.WithValue(c.Context(), constants.SUBSCRIPTION_AUTH_KEY, constants.SubscriptionAuth{
		Claims: claims,
		Token:  token,
	})

	c.SetContext(ctx)

	return c.Next()
}
