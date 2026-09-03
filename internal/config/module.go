package config

import (
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"config",
	env.Provide,
)
