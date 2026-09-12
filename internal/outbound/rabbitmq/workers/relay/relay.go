package relay

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	bea "github.com/rickferrdev/bea-go"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/publisher"
	"github.com/rickferrdev/sublyra-api/internal/outbound/rabbitmq/topology"
	"go.uber.org/fx"
)

type Worker struct {
	log          *bea.Logger
	maxAttempts  int
	batchSize    int
	pollInterval time.Duration
	cancel       context.CancelFunc
	database     subscription.Interface
	publisher    publisher.Interface
	env          *env.Env
	wg           *sync.WaitGroup
	lifecycle    fx.Lifecycle
}

var Provide = fx.Provide(New)

var Invoke = fx.Invoke(func(worker *Worker) error {
	return worker.Start()
})

type FxParams struct {
	fx.In
	fx.Lifecycle
	Log       *bea.Logger
	Env       *env.Env
	Database  subscription.Interface
	Publisher publisher.Interface
}

func New(params FxParams) *Worker {
	worker := Worker{
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
			pollInterval, err := time.ParseDuration(worker.env.OutboxPollInterval)
			if err != nil {
				return err
			}
			if pollInterval <= 0 {
				pollInterval = 5
			}
			batchSize, err := strconv.Atoi(worker.env.OutboxBatchSize)
			if err != nil {
				return err
			}
			if batchSize <= 0 {
				batchSize = 5
			}
			maxAttempts, err := strconv.Atoi(worker.env.OutboxMaxAttempts)
			if err != nil {
				return err
			}
			if maxAttempts <= 0 {
				maxAttempts = 5
			}
			worker.pollInterval = pollInterval
			worker.batchSize = batchSize
			worker.maxAttempts = maxAttempts
			worker.wg.Go(func() {
				worker.RunTicker(wctx)
			})
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if worker.cancel != nil {
				worker.cancel()
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

func (worker *Worker) RunTicker(ctx context.Context) {
	ticker := time.NewTicker(worker.pollInterval)
	defer ticker.Stop()
	for {
		if err := worker.ProcessOnce(ctx); err != nil {
			_ = worker.log.ErrorContext(ctx, "outbox relay cycle failed", bea.Any("error", err))
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (worker *Worker) ProcessOnce(ctx context.Context) error {
	for index := 0; index < worker.batchSize; index++ {
		outbox, err := worker.database.ClaimPendingOutbox(ctx, worker.maxAttempts)
		if err != nil {
			return ports.Internal(err)
		}
		if outbox == nil {
			break
		}
		if !outbox.CanIncrementAttempts(worker.maxAttempts) {
			if err = worker.MarkFailed(ctx, outbox.ID, "maximum number of attempts reached"); err != nil {
				_ = worker.log.ErrorContext(ctx, "failed to mark outbox as failed", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
			}
			continue
		}
		integrationEvent := domain.IntegrationEvent{
			EventID:     outbox.ID,
			EventType:   outbox.Event,
			AggregateID: outbox.AggregateID,
			OccurredAt:  outbox.CreatedAt,
			Payload:     outbox.Payload,
		}
		payload, err := json.Marshal(integrationEvent)
		if err != nil {
			if err = worker.MarkFailed(ctx, outbox.ID, "error converting to JSON"); err != nil {
				_ = worker.log.ErrorContext(ctx, "failed to mark outbox as failed", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
			}
			continue
		}
		var router topology.RouteConfig
		switch outbox.Event {
		case domain.EventOutboxSubscriptionCancellationRequested:
			router = topology.RouteSubscriptionCancellation
		case domain.EventOutboxSubscriptionConfirmationRequested:
			router = topology.RouteSubscriptionConfirmation
		default:
			if err = worker.MarkFailed(ctx, outbox.ID, "unknown status"); err != nil {
				_ = worker.log.ErrorContext(ctx, "failed to mark outbox as failed", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
			}
			continue
		}
		if err := worker.database.MarkProcessing(ctx, outbox.ID); err != nil {
			if err = worker.IncrementAttempts(ctx, outbox.ID, err.Error()); err != nil {
				_ = worker.log.ErrorContext(ctx, "failed to increment outbox attempts", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
			}
			continue
		}
		if err := worker.publisher.Publish(ctx, publisher.Message{
			Route:   router,
			Payload: payload,
		}); err != nil {
			err = worker.IncrementAttempts(ctx, outbox.ID, err.Error())
			_ = worker.log.ErrorContext(ctx, "failed to publish outbox event", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
			continue
		}
		err = worker.MarkPublished(ctx, outbox.ID)
		if err != nil {
			_ = worker.log.ErrorContext(ctx, "failed to mark outbox as published", bea.String("outbox_id", outbox.ID), bea.Any("error", err))
		}
	}
	return nil
}

func (worker *Worker) MarkFailed(ctx context.Context, id string, reason string) error {
	if err := worker.database.MarkFailed(ctx, id, reason); err != nil {
		return ports.Internal(err)
	}
	return nil
}

func (worker *Worker) IncrementAttempts(ctx context.Context, id string, reason string) error {
	if err := worker.database.IncrementAttempts(ctx, id, reason); err != nil {
		return ports.Internal(err)
	}
	return nil
}

func (worker *Worker) MarkPublished(ctx context.Context, id string) error {
	if err := worker.database.MarkPublished(ctx, id, time.Now()); err != nil {
		return ports.Internal(err)
	}
	return nil
}
