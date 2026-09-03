package controllers

import (
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/controllers/subscription"
	"go.uber.org/fx"
)

var Module = fx.Module("controllers",
	subscription.Invoke,
)
