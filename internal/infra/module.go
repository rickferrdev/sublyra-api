package infra

import (
	"github.com/rickferrdev/sublyra-api/internal/infra/logger"
	"github.com/rickferrdev/sublyra-api/internal/infra/mailer"
	"github.com/rickferrdev/sublyra-api/internal/infra/mongo"
	"github.com/rickferrdev/sublyra-api/internal/infra/server"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"infrastructure",
	server.Provide,
	server.Invoke,
	mongo.Provide,

	logger.Provide,
	mailer.Provide,
)
