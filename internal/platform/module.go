package platform

import (
	"github.com/rickferrdev/sublyra-api/internal/platform/jwttoken"
	"github.com/rickferrdev/sublyra-api/internal/platform/validator"
	"go.uber.org/fx"
)

var Module = fx.Module("platform",
	jwttoken.Provide,
	validator.Provide,
)
