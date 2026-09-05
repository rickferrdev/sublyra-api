package inbound

import (
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"inbound",
	rest.Module,
)
