package middlewares

import (
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/middlewares/guard"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"rest",
	guard.Invoke,
)
