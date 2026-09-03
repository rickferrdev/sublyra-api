package outbound

import (
	"github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"outbound",
	repositories.Module,
)
