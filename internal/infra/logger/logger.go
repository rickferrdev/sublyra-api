package logger

import (
	bea "github.com/rickferrdev/bea-go"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

func New() (*bea.Logger, error) {
	return bea.NewDefault(
		bea.Redact("confirmation_token", "[REDACTED]"),
		bea.Redact("unsubscribe_token", "[REDACTED]"),
		bea.Redact("token", "[REDACTED]"),
	), nil
}
