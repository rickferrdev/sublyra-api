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
