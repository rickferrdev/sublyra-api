package services

import (
	"github.com/rickferrdev/sublyra-api/internal/core/services/subscription"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"services",
	subscription.Provide,
)
