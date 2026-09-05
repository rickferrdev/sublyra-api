package publisher

import (
	"context"
	"encoding/json"
	"errors"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"go.uber.org/fx"
)

type Publisher struct {
	Client *rabbitmq.Client
}

var Provide = fx.Provide(fx.Annotate(New, fx.As(new(Interface))))

type Interface interface {
	Publish(ctx context.Context, message Message) error
}

type FxParams struct {
	fx.In
	Client *rabbitmq.Client
}

func New(params FxParams) *Publisher {
	publisher := Publisher{
		Client: params.Client,
	}

	return &publisher
}

type Message struct {
	Route   topology.RouteConfig
	Payload []byte
}

func (publisher *Publisher) Publish(ctx context.Context, message Message) error {
	if !json.Valid(message.Payload) {
		return rabbitmq.RabbitMQPublishError(errors.New("json invalid"))
	}
	if err := publisher.Client.Channel.PublishWithContext(
		ctx,
		string(message.Route.Exchange),
		string(message.Route.RoutingKey),
		false, false, amqp.Publishing{
			ContentType:  "application/json",
			Body:         message.Payload,
			DeliveryMode: amqp.Persistent,
		},
	); err != nil {
		return rabbitmq.RabbitMQPublishError(err)
	}
	return nil
}
