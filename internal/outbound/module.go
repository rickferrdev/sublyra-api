package outbound

import (
	"github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/publisher"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"outbound",
	repositories.Module,
	rabbitmq.Provide,
	rabbitmq.Invoke,
	topology.Invoke,
	publisher.Provide,
)
