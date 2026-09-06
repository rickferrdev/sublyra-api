package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"

	amqp "github.com/rabbitmq/amqp091-go"
	bea "github.com/rickferrdev/bea-go"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/infra/mailer"
	"github.com/rickferrdev/sublyra-api/internal/infra/rabbitmq"
	"github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/publisher"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"go.uber.org/fx"
)

type Worker struct {
	channel   *amqp.Channel
	log       *bea.Logger
	cancel    context.CancelFunc
	database  subscription.Interface
	mailer    mailer.Interface
	publisher publisher.Interface
	env       *env.Env
	wg        *sync.WaitGroup
	lifecycle fx.Lifecycle
	client    *rabbitmq.Client
}

var Provide = fx.Provide(New)

var Invoke = fx.Invoke(func(worker *Worker) error {
	return worker.Start()
})

type FxParams struct {
	fx.In
	fx.Lifecycle
	Mailer    mailer.Interface
	Client    *rabbitmq.Client
	Log       *bea.Logger
	Env       *env.Env
	Database  subscription.Interface
	Publisher publisher.Interface
}

func New(params FxParams) *Worker {
	worker := Worker{
		mailer:    params.Mailer,
		client:    params.Client,
		log:       params.Log,
		database:  params.Database,
		publisher: params.Publisher,
		env:       params.Env,
		lifecycle: params.Lifecycle,
	}
	return &worker
}

func (worker *Worker) Start() error {
	wctx, cancel := context.WithCancel(context.Background())
	worker.cancel = cancel
	worker.wg = &sync.WaitGroup{}
	worker.lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			prefetch, err := strconv.Atoi(worker.env.RabbitMQPrefetch)
			if err != nil {
				return err
			}
			if prefetch <= 0 {
				prefetch = 5
			}
			channel, err := worker.client.Connection.Channel()
			if err != nil {
				return err
			}
			worker.channel = channel
			if err := worker.channel.Qos(prefetch, 0, false); err != nil {
				return err
			}
			deliveries, err := worker.channel.Consume(
				string(topology.QueueEmail),
				"sublyra.email.consumer",
				false, false, false, false, nil,
			)
			if err != nil {
				return err
			}
			worker.wg.Go(func() {
				worker.process(wctx, deliveries)
			})
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if worker.cancel != nil {
				worker.cancel()
			}
			if worker.channel != nil {
				if err := worker.channel.Close(); err != nil {
					return err
				}
			}
			done := make(chan struct{})
			go func() {
				worker.wg.Wait()
				close(done)
			}()
			select {
			case <-done:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
	})
	return nil
}

func (worker *Worker) process(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for delivery := range deliveries {
		var payload domain.IntegrationEvent
		if err := json.Unmarshal(delivery.Body, &payload); err != nil {
			err = worker.SendToDlq(ctx, delivery, payload, "invalid payload")
			_ = worker.log.ErrorContext(ctx, "failed to handle invalid email payload", bea.Any("error", err))
			continue
		}
		recipient, ok := payload.Payload["email"].(string)
		if !ok {
			err := worker.SendToDlq(ctx, delivery, payload, "invalid payload")
			_ = worker.log.ErrorContext(ctx, "email payload missing recipient", bea.String("event_id", payload.EventID), bea.Any("error", err))
			continue
		}
		if recipient == "" {
			err := worker.SendToDlq(ctx, delivery, payload, "invalid payload")
			_ = worker.log.ErrorContext(ctx, "email payload has empty recipient", bea.String("event_id", payload.EventID), bea.Any("error", err))
			continue
		}
		switch payload.EventType {
		case domain.EventOutboxSubscriptionCancellationRequested:
			unsubscribeToken, ok := payload.Payload["unsubscribe_token"].(string)
			if !ok {
				err := worker.SendToDlq(ctx, delivery, payload, "invalid unsubscribe token")
				_ = worker.log.ErrorContext(ctx, "email payload missing unsubscribe token", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
			if unsubscribeToken == "" {
				err := worker.SendToDlq(ctx, delivery, payload, "invalid confirmation token")
				_ = worker.log.ErrorContext(ctx, "email payload has empty unsubscribe token", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
			email := worker.mailer.EmailCancellation(mailer.CancellationData{
				Recipients:      []string{recipient},
				Subject:         "Subscription cancellation",
				CancellationURL: worker.MountUrl("unsubscription", unsubscribeToken),
				Email:           recipient,
			})
			if err := worker.SendEmail(ctx, email); err != nil {
				err = worker.SendToRetry(ctx, delivery, payload, err.Error())
				_ = worker.log.ErrorContext(ctx, "failed to send cancellation email", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
		case domain.EventOutboxSubscriptionConfirmationRequested:
			confirmationToken, ok := payload.Payload["confirmation_token"].(string)
			if !ok {
				err := worker.SendToDlq(ctx, delivery, payload, "invalid confirmation token")
				_ = worker.log.ErrorContext(ctx, "email payload missing confirmation token", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
			if confirmationToken == "" {
				err := worker.SendToDlq(ctx, delivery, payload, "invalid confirmation token")
				_ = worker.log.ErrorContext(ctx, "email payload has empty confirmation token", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
			email := worker.mailer.EmailConfirmation(mailer.ConfirmationData{
				Recipients:      []string{recipient},
				Subject:         "Subscription confirmation",
				ConfirmationURL: worker.MountUrl("subscription", confirmationToken),
				Email:           recipient,
			})
			if err := worker.SendEmail(ctx, email); err != nil {
				err = worker.SendToRetry(ctx, delivery, payload, err.Error())
				_ = worker.log.ErrorContext(ctx, "failed to send confirmation email", bea.String("event_id", payload.EventID), bea.Any("error", err))
				continue
			}
		default:
			err := worker.SendToDlq(ctx, delivery, payload, "invalid payload")
			_ = worker.log.ErrorContext(ctx, "unknown email event type", bea.String("event_id", payload.EventID), bea.String("event_type", string(payload.EventType)), bea.Any("error", err))
			continue
		}
		if err := worker.database.MarkDelivered(ctx, payload.EventID); err != nil {
			_ = worker.log.ErrorContext(ctx, "failed to mark outbox as delivered", bea.String("event_id", payload.EventID), bea.Any("error", err))
			_ = delivery.Ack(false)
			continue
		}
		_ = delivery.Ack(false)
	}
}

func (worker *Worker) MountUrl(action string, token string) string {
	url := url.URL{
		Scheme: "https",
		Host:   "api.examples.com",
		Path:   fmt.Sprintf("/%s/confirm", action),
	}
	q := url.Query()
	q.Set("token", token)
	url.RawQuery = q.Encode()
	return url.String()
}

func (worker *Worker) SendEmail(ctx context.Context, email mailer.Email) error {
	if err := worker.mailer.Send(ctx, email); err != nil {
		return ports.Internal(err)
	}
	return nil
}

func (worker *Worker) MarkFailed(ctx context.Context, delivery amqp.Delivery, id string, reason string) error {
	if err := worker.database.MarkFailed(ctx, id, reason); err != nil {
		return ports.Internal(err)
	}
	return nil
}

func (worker *Worker) SendToDlq(ctx context.Context, delivery amqp.Delivery, payload domain.IntegrationEvent, reason string) error {
	bytes, err := json.Marshal(payload)
	if err != nil {
		return ports.Internal(err)
	}
	if err := worker.publisher.Publish(ctx, publisher.Message{
		Route:   topology.RouteSubscriptionDead,
		Payload: bytes,
	}); err != nil {
		return ports.Internal(err)
	}
	_ = delivery.Ack(false)
	if err := worker.MarkFailed(ctx, delivery, payload.EventID, reason); err != nil {
		return err
	}
	return nil
}

func (worker *Worker) SendToRetry(ctx context.Context, delivery amqp.Delivery, payload domain.IntegrationEvent, reason string) error {
	if err := worker.database.IncrementAttempts(ctx, payload.EventID, reason); err != nil {
		return ports.Internal(err)
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return ports.Internal(err)
	}
	if err := worker.publisher.Publish(ctx, publisher.Message{Route: topology.RouteRetryQueue, Payload: bytes}); err != nil {
		return ports.Internal(err)
	}
	return delivery.Ack(false)
}
