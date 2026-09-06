package rabbitmq

import (
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/publisher"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/workers"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"rabbitmq",
	topology.Invoke,
	publisher.Provide,
	workers.Module,
)
