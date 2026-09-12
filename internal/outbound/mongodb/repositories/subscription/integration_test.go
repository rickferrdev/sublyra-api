//go:build integration

package subscription_test

import (
	"context"
	"testing"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	dbsubscription "github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func TestMongoDB_Integration(t *testing.T) {
	ctx := context.Background()

	mongodbContainer, err := mongodb.Run(ctx, "mongo:8.0",
		mongodb.WithReplicaSet("rs0"),
	)
	if err != nil {
		t.Fatalf("failed to start mongodb container: %s", err)
	}
	t.Cleanup(func() {
		if err := mongodbContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	})

	endpoint, err := mongodbContainer.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	client, err := mongo.Connect(options.Client().ApplyURI(endpoint).SetDirect(true))
	if err != nil {
		t.Fatalf("failed to connect to mongodb: %s", err)
	}
	t.Cleanup(func() {
		_ = client.Disconnect(ctx)
	})

	repo := dbsubscription.New(dbsubscription.FxParams{
		Client: client,
	})

	t.Run("InsertWithOutbox transactionally creates subscription and outbox event", func(t *testing.T) {
		email := "integration@example.com"
		sub := domain.Subscription{
			ID:                bson.NewObjectID().Hex(),
			Email:             email,
			Status:            domain.SubscriptionStatusPending,
			ConfirmationToken: "token_abc_123",
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		}

		err := repo.InsertWithOutbox(ctx, domain.EventOutboxSubscriptionConfirmationRequested, sub)
		if err != nil {
			t.Fatalf("expected InsertWithOutbox to succeed, got: %v", err)
		}

		// Verify subscription document
		foundSub, err := repo.Find(ctx, email)
		if err != nil || foundSub == nil {
			t.Fatalf("expected to find subscription, got err: %v", err)
		}
		if foundSub.Email != email {
			t.Errorf("expected email %s, got %s", email, foundSub.Email)
		}

		// Verify outbox document
		outbox, err := repo.FindOutbox(ctx, domain.EventOutboxSubscriptionConfirmationRequested, email)
		if err != nil || outbox == nil {
			t.Fatalf("expected to find outbox event, got err: %v", err)
		}
		if outbox.Status != domain.OutboxSubscriptionStatusPending {
			t.Errorf("expected status pending, got %s", outbox.Status)
		}
	})

	t.Run("ClaimPendingOutbox atomically claims and updates outbox status to processing", func(t *testing.T) {
		claimed, err := repo.ClaimPendingOutbox(ctx, 5)
		if err != nil {
			t.Fatalf("expected ClaimPendingOutbox to succeed, got: %v", err)
		}
		if claimed == nil {
			t.Fatal("expected a claimed outbox event, got nil")
		}
		if claimed.Status != domain.OutboxSubscriptionStatusProcessing {
			t.Errorf("expected status processing, got %s", claimed.Status)
		}
	})
}
