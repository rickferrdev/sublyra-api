package middlewares

import (
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/middlewares/guardtoken"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"middlewares",
	guardtoken.Provide,
)
