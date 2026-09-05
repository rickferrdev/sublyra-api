package rabbitmq

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"go.uber.org/fx"
)

var Invoke = fx.Invoke(func(rabbit *RabbitMQ) error {
	return rabbit.Invoke()
})

var Provide = fx.Provide(
	New,
	(*RabbitMQ).Provide,
)

type FxParams struct {
	fx.In
	Env       *env.Env
	Lifecycle fx.Lifecycle
}

type RabbitMQ struct {
	client    *Client
	env       *env.Env
	lifecycle fx.Lifecycle
}

func New(params FxParams) (*RabbitMQ, error) {
	rabbit := RabbitMQ{
		client:    &Client{},
		env:       params.Env,
		lifecycle: params.Lifecycle,
	}
	return &rabbit, nil
}

type Client struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

func (rabbit *RabbitMQ) Provide() *Client {
	return rabbit.client
}

func (rabbit *RabbitMQ) Invoke() error {
	rabbit.lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			connection, err := amqp.Dial(rabbit.env.RabbitMQURI)
			if err != nil {
				return err
			}
			channel, err := connection.Channel()
			if err != nil {
				_ = connection.Close()
				return err
			}
			rabbit.client.Connection = connection
			rabbit.client.Channel = channel
			return nil
		},
		OnStop: func(ctx context.Context) error {
			var channelErr error
			if rabbit.client.Channel != nil {
				channelErr = rabbit.client.Channel.Close()
			}
			if rabbit.client.Connection != nil {
				if err := rabbit.client.Connection.Close(); err != nil {
					return err
				}
			}
			return channelErr
		},
	})
	return nil
}
