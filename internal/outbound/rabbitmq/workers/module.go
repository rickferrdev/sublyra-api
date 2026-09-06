package workers

import (
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/workers/email"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/workers/relay"
	"go.uber.org/fx"
)

var Module = fx.Module(
	"workers",
	relay.Provide,
	relay.Invoke,
	email.Provide,
	email.Invoke,
)
