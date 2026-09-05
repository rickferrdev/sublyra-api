// Modules are grouped by responsibility.
//
// config: application configuration and environment variables.
// infra: framework-level infrastructure, such as HTTP server and logger.
// outbound: adapters that communicate with external systems.
// services: application use cases / business services.
// inbound: adapters that receive input, such as HTTP handlers.
package main

import (
	"github.com/rickferrdev/sublyra-api/internal/config"
	"github.com/rickferrdev/sublyra-api/internal/core/services"
	"github.com/rickferrdev/sublyra-api/internal/inbound"
	"github.com/rickferrdev/sublyra-api/internal/infra"
	"github.com/rickferrdev/sublyra-api/internal/outbound"
	"github.com/rickferrdev/sublyra-api/internal/platform"
	"go.uber.org/fx"
)

func main() {
	fx.New(
		config.Module,
		platform.Module,
		infra.Module,
		outbound.Module,
		services.Module,
		inbound.Module,
	).Run()
}
