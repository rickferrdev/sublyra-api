package domain

import (
	"fmt"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/core/ports"
)

type (
	SubscriptionStatus string

	OutboxSubscriptionEvent  string
	OutboxSubscriptionStatus string
)

const (
	EventOutboxSubscriptionConfirmationRequested OutboxSubscriptionEvent = "outbox_subscription_confirmation_requested"
	EventOutboxSubscriptionCancellationRequested OutboxSubscriptionEvent = "outbox_subscription_cancellation_requested"

	OutboxSubscriptionStatusPending   OutboxSubscriptionStatus = "pending"
	OutboxSubscriptionStatusPublished OutboxSubscriptionStatus = "published"
	OutboxSubscriptionStatusFailed    OutboxSubscriptionStatus = "failed"

	SubscriptionStatusPending      SubscriptionStatus = "pending"
	SubscriptionStatusSubscribed   SubscriptionStatus = "subscribed"
	SubscriptionStatusUnsubscribed SubscriptionStatus = "unsubscribed"
)

type Subscription struct {
	ID     string             `json:"id"`
	Email  string             `json:"email"`
	Status SubscriptionStatus `json:"status"`

	SubscribedAt   time.Time `json:"subscribed_at"`
	UnsubscribedAt time.Time `json:"unsubscribed_at"`

	ConfirmationToken string `json:"confirmation_token"`
	UnsubscribeToken  string `json:"unsubscribe_token"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OutboxSubscription struct {
	ID          string                   `json:"id"`
	AggregateID string                   `json:"aggregate_id"`
	Email       string                   `json:"email"`
	Event       OutboxSubscriptionEvent  `json:"event"`
	Attempts    int                      `json:"attempts"`
	Payload     map[string]any           `json:"payload"`
	Status      OutboxSubscriptionStatus `json:"status"`

	PublishedAt time.Time `json:"published_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (subscription *Subscription) IsSubscribed() bool {
	return subscription.Status == SubscriptionStatusSubscribed
}

func (subscription *Subscription) IsUnsubscribed() bool {
	return subscription.Status == SubscriptionStatusUnsubscribed
}

func (subscription *Subscription) IsPending() bool {
	return subscription.Status == SubscriptionStatusPending
}

func (subscription *Subscription) Subscribe() {
	subscription.Status = SubscriptionStatusSubscribed
	subscription.SubscribedAt = time.Now()
	subscription.UnsubscribedAt = time.Time{}
}

func (subscription *Subscription) Unsubscribe() {
	subscription.Status = SubscriptionStatusUnsubscribed
	subscription.UnsubscribedAt = time.Now()
}

func (subscription *Subscription) CompareTokenConfirmation(token string) bool {
	return subscription.ConfirmationToken == token
}

func (subscription *Subscription) CompareTokenUnsubscribe(token string) bool {
	return subscription.UnsubscribeToken == token
}

func (outbox *OutboxSubscription) IsPending() bool {
	return outbox.Event == EventOutboxSubscriptionConfirmationRequested
}

func (outbox *OutboxSubscription) IsCompleted() bool {
	return outbox.Event == EventOutboxSubscriptionCancellationRequested
}

func (outbox *OutboxSubscription) MarkAsCompleted() {
	outbox.Event = EventOutboxSubscriptionCancellationRequested
}

func (outbox *OutboxSubscription) IncrementAttempts() {
	outbox.Attempts++
}

func (outbox *OutboxSubscription) MarkAsPending() {
	outbox.Event = EventOutboxSubscriptionConfirmationRequested
}

func (outbox *OutboxSubscription) CanIncrementAttempts(maxAttempts int) bool {
	return outbox.Attempts < maxAttempts
}

func (subscription *Subscription) CooldownUnsubscription() error {
	availableAt := subscription.UnsubscribedAt.Add(1 * time.Second)
	if time.Now().Before(availableAt) {
		return ports.Conflict(fmt.Errorf("subscription can only renewed after %s", availableAt.Format((time.RFC3339))))
	}
	return nil
}
