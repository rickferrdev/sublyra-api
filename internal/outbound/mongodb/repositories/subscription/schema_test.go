package subscription

import (
	"testing"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNewOutboxSubscriptionSchema(t *testing.T) {
	subscription := domain.Subscription{ID: bson.NewObjectID().Hex(), Email: "person@example.com", Status: domain.SubscriptionStatusPending, ConfirmationToken: "confirmation-token", UnsubscribeToken: "unsubscribe-token"}
	document, err := NewOutboxSubscriptionSchema(subscription, domain.EventOutboxSubscriptionConfirmationRequested)
	if err != nil {
		t.Fatalf("create confirmation outbox: %v", err)
	}
	if document.Status != domain.OutboxSubscriptionStatusPending || document.Attempts != 0 {
		t.Fatalf("unexpected initial outbox state: %#v", document)
	}
	if document.Payload["confirmation_token"] != "confirmation-token" || document.CreatedAt.IsZero() || document.UpdatedAt.IsZero() {
		t.Fatalf("confirmation outbox is incomplete: %#v", document)
	}
	cancellation, err := NewOutboxSubscriptionSchema(subscription, domain.EventOutboxSubscriptionCancellationRequested)
	if err != nil {
		t.Fatalf("create cancellation outbox: %v", err)
	}
	if cancellation.Payload["unsubscribe_token"] != "unsubscribe-token" {
		t.Fatalf("expected unsubscribe token, got %#v", cancellation.Payload)
	}
	if _, exists := cancellation.Payload["confirmation_token"]; exists {
		t.Fatal("cancellation payload must not include confirmation token")
	}
}

func TestOutboxSubscriptionSchemaToDomainPreservesRelayFields(t *testing.T) {
	id, aggregateID := bson.NewObjectID(), bson.NewObjectID()
	now := time.Now().UTC().Truncate(time.Millisecond)
	schema := OutboxSubscriptionSchema{ID: id, AggregateID: aggregateID, Email: "person@example.com", Event: domain.EventOutboxSubscriptionConfirmationRequested, Status: domain.OutboxSubscriptionStatusFailed, Attempts: 3, LastError: "broker unavailable", CreatedAt: now, UpdatedAt: now}
	actual, err := schema.ToDomain()
	if err != nil {
		t.Fatalf("convert schema: %v", err)
	}
	if actual.ID != id.Hex() || actual.AggregateID != aggregateID.Hex() || actual.LastError != schema.LastError || actual.Attempts != 3 {
		t.Fatalf("relay fields were not preserved: %#v", actual)
	}
}
