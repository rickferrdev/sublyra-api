package subscription

import (
	"errors"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type (
	SubscriptionSchema struct {
		ID     bson.ObjectID             `bson:"_id"`
		Email  string                    `bson:"email"`
		Status domain.SubscriptionStatus `bson:"status"`

		SubscribedAt   time.Time `bson:"subscribed_at,omitempty"`
		UnsubscribedAt time.Time `bson:"unsubscribed_at,omitempty"`

		ConfirmationToken string `bson:"confirmation_token,omitempty"`
		UnsubscribeToken  string `bson:"unsubscribe_token,omitempty"`

		CreatedAt time.Time `bson:"created_at"`
		UpdatedAt time.Time `bson:"updated_at"`
	}

	SubscriptionUpdateSchema struct {
		Email  *string                    `bson:"email,omitempty"`
		Status *domain.SubscriptionStatus `bson:"status,omitempty"`

		SubscribedAt   *time.Time `bson:"subscribed_at,omitempty"`
		UnsubscribedAt *time.Time `bson:"unsubscribed_at,omitempty"`

		ConfirmationToken *string `bson:"confirmation_token,omitempty"`
		UnsubscribeToken  *string `bson:"unsubscribe_token,omitempty"`

		CreatedAt *time.Time `bson:"created_at,omitempty"`
		UpdatedAt *time.Time `bson:"updated_at,omitempty"`

		Unset []string `bson:"-"`
	}

	OutboxSubscriptionSchema struct {
		ID          bson.ObjectID                   `bson:"_id"`
		AggregateID bson.ObjectID                   `bson:"aggregate_id"`
		Email       string                          `bson:"email"`
		Event       domain.OutboxSubscriptionEvent  `bson:"event"`
		Attempts    int                             `bson:"attempts"`
		Payload     map[string]any                  `bson:"payload"`
		Status      domain.OutboxSubscriptionStatus `bson:"status"`

		PublishedAt time.Time `bson:"published_at,omitempty"`

		CreatedAt time.Time `bson:"created_at"`
		UpdatedAt time.Time `bson:"updated_at"`
	}
)

// NewSubscriptionSchema converts a domain subscription into its MongoDB schema.
//
// It may return ObjectIDInvalid when subscription.ID is not a valid BSON
// ObjectID.
func NewSubscriptionSchema(subscription domain.Subscription) (*SubscriptionSchema, error) {
	ID, err := bson.ObjectIDFromHex(subscription.ID)
	if err != nil {
		return nil, ObjectIDInvalidError(err)
	}
	return &SubscriptionSchema{
		ID:                ID,
		Email:             subscription.Email,
		Status:            subscription.Status,
		SubscribedAt:      subscription.SubscribedAt,
		UnsubscribedAt:    subscription.UnsubscribedAt,
		ConfirmationToken: subscription.ConfirmationToken,
		UnsubscribeToken:  subscription.UnsubscribeToken,
		CreatedAt:         subscription.CreatedAt,
		UpdatedAt:         subscription.UpdatedAt,
	}, nil
}

// NewOutboxSubscriptionSchema creates an outbox schema for a subscription event.
//
// It may return ObjectIDInvalid when subscription.ID is not a valid BSON
// ObjectID.
func NewOutboxSubscriptionSchema(subscription domain.Subscription, event domain.OutboxSubscriptionEvent) (*OutboxSubscriptionSchema, error) {
	aggregateID, err := bson.ObjectIDFromHex(subscription.ID)
	if err != nil {
		return nil, ObjectIDInvalidError(err)
	}
	payload := map[string]any{
		"status": subscription.Status,
		"email":  subscription.Email,
	}
	switch event {
	case domain.EventOutboxSubscriptionConfirmationRequested:
		payload["confirmation_token"] = subscription.ConfirmationToken
	case domain.EventOutboxSubscriptionCancellationRequested:
		payload["unsubscribe_token"] = subscription.UnsubscribeToken
	}
	return &OutboxSubscriptionSchema{
		ID:          bson.NewObjectID(),
		AggregateID: aggregateID,
		Email:       subscription.Email,
		Event:       event,
		Status:      domain.OutboxSubscriptionStatusPending,
		Attempts:    0,
		Payload:     payload,
		PublishedAt: time.Time{},
		CreatedAt:   subscription.CreatedAt,
		UpdatedAt:   subscription.UpdatedAt,
	}, nil
}

// ToDomain converts a subscription MongoDB schema into its domain model.
//
// It may return ObjectIDInvalid when the schema has an empty ID.
func (schema *SubscriptionSchema) ToDomain() (*domain.Subscription, error) {
	if schema.ID.IsZero() {
		return nil, ObjectIDInvalidError(errors.New("object id is empty"))
	}
	return &domain.Subscription{
		ID:                schema.ID.Hex(),
		Email:             schema.Email,
		Status:            schema.Status,
		SubscribedAt:      schema.SubscribedAt,
		UnsubscribedAt:    schema.UnsubscribedAt,
		ConfirmationToken: schema.ConfirmationToken,
		UnsubscribeToken:  schema.UnsubscribeToken,
		CreatedAt:         schema.CreatedAt,
		UpdatedAt:         schema.UpdatedAt,
	}, nil
}

// ToDomain converts an outbox MongoDB schema into its domain model.
//
// It may return ObjectIDInvalid when the schema has an empty ID or AggregateID.
func (schema *OutboxSubscriptionSchema) ToDomain() (*domain.OutboxSubscription, error) {
	if schema.ID.IsZero() {
		return nil, ObjectIDInvalidError(errors.New("object id is empty"))
	}
	if schema.AggregateID.IsZero() {
		return nil, ObjectIDInvalidError(errors.New("object id is empty"))
	}
	return &domain.OutboxSubscription{
		ID:          schema.ID.Hex(),
		AggregateID: schema.AggregateID.Hex(),
		Email:       schema.Email,
		Event:       schema.Event,
		Attempts:    schema.Attempts,
		Payload:     schema.Payload,
		Status:      schema.Status,
		PublishedAt: schema.PublishedAt,
		CreatedAt:   schema.CreatedAt,
		UpdatedAt:   schema.UpdatedAt,
	}, nil
}
