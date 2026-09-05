package repositories

import (
	"github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"go.uber.org/fx"
)

var Module = fx.Module("repositories",
	subscription.Provide,
	subscription.Invoke,
)
