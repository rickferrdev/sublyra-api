package rest

import (
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/controllers"
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/middlewares"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"rest",
	controllers.Module,
	middlewares.Module,
)
