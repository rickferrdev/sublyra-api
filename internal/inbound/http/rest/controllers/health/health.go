package health

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rickferrdev/bea-go"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/infra/rabbitmq"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.uber.org/fx"
)

var Invoke = fx.Invoke(New)

type Controller struct {
	duration    time.Duration
	logger      *bea.Logger
	env         *env.Env
	router      fiber.Router
	clientMongo *mongo.Client
	clientAMQP  *rabbitmq.Client
}

type FxParams struct {
	fx.In
	Logger *bea.Logger
	Env    *env.Env
	Router fiber.Router
	Mongo  *mongo.Client
	AMQP   *rabbitmq.Client
}

func New(params FxParams) *Controller {
	duration, err := time.ParseDuration(params.Env.HealthTimeout)
	if err != nil {
		duration = time.Second * 5
		params.Logger.Error("error configuring the timeout for the /health route", bea.String("error", err.Error()))
	}
	controller := &Controller{
		duration:    duration,
		router:      params.Router,
		clientMongo: params.Mongo,
		clientAMQP:  params.AMQP,
	}

	params.Router.Get("/health", controller.Health)
	return controller
}

func (controller *Controller) Health(c fiber.Ctx) error {
	services, allHealthy := make(map[string]ServiceDTO), true
	ctx, cancel := context.WithTimeout(c.Context(), controller.duration)
	defer cancel()
	if err := controller.clientMongo.Ping(ctx, readpref.PrimaryPreferred()); err != nil {
		services["mongodb"] = ServiceDTO{
			Status: "down",
		}
		allHealthy = false
	} else {
		services["mongodb"] = ServiceDTO{
			Status: "up",
		}
	}
	if controller.clientAMQP == nil || controller.clientAMQP.Connection.IsClosed() {
		services["rabbitmq"] = ServiceDTO{
			Status: "down",
		}
		allHealthy = false
	} else {
		services["rabbitmq"] = ServiceDTO{
			Status: "up",
		}
	}
	status := "ok"
	httpCode := fiber.StatusOK
	if !allHealthy {
		status = "degraded"
		httpCode = fiber.StatusServiceUnavailable
	}
	return c.Status(httpCode).JSON(ResponseHealthDTO{
		Status:   status,
		Services: services,
	})
}
