package topology

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq"
	"go.uber.org/fx"
)

var Invoke = fx.Invoke(register)

type (
	RoutingKey   string
	ExchangeType string
	ExchangeName string
	QueueName    string
	Table        map[string]any
)

type ExchangeConfig struct {
	Name       ExchangeName
	Type       ExchangeType
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       Table
}

type QueueConfig struct {
	Name       QueueName
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       Table
}

type RouteConfig struct {
	Exchange   ExchangeName
	RoutingKey RoutingKey
}

type BindingConfig struct {
	Route  RouteConfig
	NoWait bool
	Args   Table
}

type Router struct {
	Exchange ExchangeConfig
	Queue    QueueConfig
	Binding  BindingConfig
}

const (
	ExchangeEvents ExchangeName = "sublyra.events"
	ExchangeRetry  ExchangeName = "sublyra.retry"
	ExchangeDead   ExchangeName = "sublyra.dlx"

	QueueEmail      QueueName = "sublyra.email"
	QueueEmailRetry QueueName = "sublyra.email.retry"
	QueueEmailDLQ   QueueName = "sublyra.email.dlq"

	RoutingSubscriptionConfirmation RoutingKey = "subscription.confirmation.requested"
	RoutingSubscriptionCancellation RoutingKey = "subscription.cancellation.requested"
	RoutingEmailRetry               RoutingKey = "subscription.email.retry"
	RoutingSubscriptionDead         RoutingKey = "subscription.dead"

	ExchangeDirect ExchangeType = "direct"

	RetryDelayMilliseconds = 30_000
)

var (
	RouteSubscriptionConfirmation = RouteConfig{
		Exchange:   ExchangeEvents,
		RoutingKey: RoutingSubscriptionConfirmation,
	}
	RouteSubscriptionCancellation = RouteConfig{
		Exchange:   ExchangeEvents,
		RoutingKey: RoutingSubscriptionCancellation,
	}
	RouteEmailRetry = RouteConfig{
		Exchange:   ExchangeEvents,
		RoutingKey: RoutingEmailRetry,
	}
	RouteRetryQueue = RouteConfig{
		Exchange:   ExchangeRetry,
		RoutingKey: RoutingEmailRetry,
	}
	RouteSubscriptionDead = RouteConfig{
		Exchange:   ExchangeDead,
		RoutingKey: RoutingSubscriptionDead,
	}
)

var (
	eventsExchange = ExchangeConfig{Name: ExchangeEvents, Type: ExchangeDirect, Durable: true}
	retryExchange  = ExchangeConfig{Name: ExchangeRetry, Type: ExchangeDirect, Durable: true}
	deadExchange   = ExchangeConfig{Name: ExchangeDead, Type: ExchangeDirect, Durable: true}

	emailQueue = QueueConfig{
		Name:    QueueEmail,
		Durable: true,
		Args: Table{
			"x-dead-letter-exchange":    string(ExchangeRetry),
			"x-dead-letter-routing-key": string(RoutingEmailRetry),
		},
	}
	retryQueue = QueueConfig{
		Name:    QueueEmailRetry,
		Durable: true,
		Args: Table{
			"x-message-ttl":             RetryDelayMilliseconds,
			"x-dead-letter-exchange":    string(ExchangeEvents),
			"x-dead-letter-routing-key": string(RoutingEmailRetry),
		},
	}
	deadQueue = QueueConfig{Name: QueueEmailDLQ, Durable: true}
)

var Topology = []Router{
	{Exchange: eventsExchange, Queue: emailQueue, Binding: BindingConfig{Route: RouteSubscriptionConfirmation}},
	{Exchange: eventsExchange, Queue: emailQueue, Binding: BindingConfig{Route: RouteSubscriptionCancellation}},
	{Exchange: eventsExchange, Queue: emailQueue, Binding: BindingConfig{Route: RouteEmailRetry}},
	{Exchange: retryExchange, Queue: retryQueue, Binding: BindingConfig{Route: RouteRetryQueue}},
	{Exchange: deadExchange, Queue: deadQueue, Binding: BindingConfig{Route: RouteSubscriptionDead}},
}

func register(client *rabbitmq.Client, lifecycle fx.Lifecycle) {
	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			return Declare(client)
		},
	})
}

func Declare(client *rabbitmq.Client) error {
	for _, router := range Topology {
		exchange := router.Exchange
		if err := client.Channel.ExchangeDeclare(
			string(exchange.Name),
			string(exchange.Type),
			exchange.Durable,
			exchange.AutoDelete,
			exchange.Internal,
			exchange.NoWait,
			amqp.Table(exchange.Args),
		); err != nil {
			return fmt.Errorf("declare exchange %q: %w", exchange.Name, err)
		}
		queue := router.Queue
		if _, err := client.Channel.QueueDeclare(
			string(queue.Name),
			queue.Durable,
			queue.AutoDelete,
			queue.Exclusive,
			queue.NoWait,
			amqp.Table(queue.Args),
		); err != nil {
			return fmt.Errorf("declare queue %q: %w", queue.Name, err)
		}
		binding := router.Binding
		if err := client.Channel.QueueBind(
			string(queue.Name),
			string(binding.Route.RoutingKey),
			string(binding.Route.Exchange),
			binding.NoWait,
			amqp.Table(binding.Args),
		); err != nil {
			return fmt.Errorf(
				"bind queue %q to exchange %q with routing key %q: %w",
				queue.Name,
				binding.Route.Exchange,
				binding.Route.RoutingKey,
				err,
			)
		}
	}

	return nil
}
