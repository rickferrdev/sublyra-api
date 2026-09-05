package domain

import "testing"

func TestOutboxSubscriptionStatusTransitions(t *testing.T) {
	outbox := OutboxSubscription{Status: OutboxSubscriptionStatusPending}
	if !outbox.IsPending() {
		t.Fatal("pending outbox must be pending")
	}
	outbox.MarkAsPublished()
	if !outbox.IsPublished() || outbox.PublishedAt.IsZero() {
		t.Fatal("published outbox must have a status and publication time")
	}
	outbox.MarkAsFailed()
	if outbox.Status != OutboxSubscriptionStatusFailed {
		t.Fatalf("expected failed status, got %q", outbox.Status)
	}
	outbox.MarkAsPending()
	if !outbox.IsPending() {
		t.Fatal("outbox must return to pending")
	}
}

func TestOutboxSubscriptionAttemptsAndCancellation(t *testing.T) {
	outbox := OutboxSubscription{Event: EventOutboxSubscriptionCancellationRequested}
	if !outbox.IsCancelled() {
		t.Fatal("cancellation event must be identified")
	}
	if !outbox.CanIncrementAttempts(1) {
		t.Fatal("zero attempts must be below the limit")
	}
	outbox.IncrementAttempts()
	if outbox.CanIncrementAttempts(1) {
		t.Fatal("attempt limit must be exclusive")
	}
}
