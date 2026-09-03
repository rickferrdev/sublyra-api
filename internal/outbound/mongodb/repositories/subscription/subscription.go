package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

var Provide = fx.Provide(fx.Annotate(New, fx.As(new(Interface))))

type Database struct {
	client        *mongo.Client
	subscriptions *mongo.Collection
	outbox        *mongo.Collection
}

type Interface interface {
	Insert(ctx context.Context, subscription domain.Subscription) error
	Find(ctx context.Context, email string) (*domain.Subscription, error)
	Update(ctx context.Context, email string, update SubscriptionUpdateSchema) error
	Delete(ctx context.Context, email string) error
	RenewConfirmation(ctx context.Context, email, token string) error
	RenewUnsubscribed(ctx context.Context, email, token string) error

	InsertOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, outbox domain.Subscription) error
	FindOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email string) (*domain.OutboxSubscription, error)

	InsertWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, subscription domain.Subscription) error
	RenewConfirmationWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, confirmation string) error
	RenewUnsubscribedWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, confirmation string) error
}

type FxParams struct {
	fx.In
	Client *mongo.Client
}

func New(params FxParams) Interface {
	subscription := params.Client.Database(SUBSCRIPTIONS_DATABASE).Collection(SUBSCRIPTIONS_COLLECTION)
	outbox := params.Client.Database(SUBSCRIPTIONS_DATABASE).Collection(OUTBOX_COLLECTION)
	database := &Database{
		client:        params.Client,
		subscriptions: subscription,
		outbox:        outbox,
	}
	return database
}

// Insert persists a subscription.
//
// It may return the following error codes:
//   - ObjectIDInvalid when subscription.ID is not a valid BSON ObjectID;
//   - CodeDuplicateExists when a subscription with the same unique key exists;
//   - ports.CodeInternal when MongoDB fails to insert the document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) Insert(ctx context.Context, subscription domain.Subscription) error {
	document, err := NewSubscriptionSchema(subscription)
	if err != nil {
		return err
	}
	res, err := database.subscriptions.InsertOne(ctx, document)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return DuplicateExistsError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	return nil
}

// Find retrieves a subscription by email.
//
// It may return the following error codes:
//   - CodeNotFound when no subscription matches the email;
//   - ports.CodeInternal when MongoDB fails to find or decode the document;
//   - ObjectIDInvalid when the stored subscription has an empty ID.
func (database *Database) Find(ctx context.Context, email string) (*domain.Subscription, error) {
	filter := bson.M{
		"email": email,
	}
	var output SubscriptionSchema
	if err := database.subscriptions.FindOne(ctx, filter).Decode(&output); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, NotFoundError(err)
		}
		return nil, InternalError(err)
	}
	return output.ToDomain()
}

// Update changes the fields of the subscription identified by email.
//
// It may return the following error codes:
//   - CodeNotFound when no subscription matches the email;
//   - ports.CodeInternal when MongoDB fails to update the document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) Update(ctx context.Context, email string, update SubscriptionUpdateSchema) error {
	filter := bson.M{
		"email": email,
	}
	document := bson.M{
		"$set": update,
	}
	if len(update.Unset) > 0 {
		unset := bson.M{}
		for _, field := range update.Unset {
			unset[field] = ""
		}
		document["$unset"] = unset
	}
	res, err := database.subscriptions.UpdateOne(ctx, filter, document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return NotFoundError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	if res.MatchedCount == 0 {
		return NotFoundError(nil)
	}
	return nil
}

// Delete removes the subscription identified by email.
//
// It may return the following error codes:
//   - CodeNotFound when MongoDB reports no document or deletes no document;
//   - ports.CodeInternal when MongoDB fails to delete the document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) Delete(ctx context.Context, email string) error {
	filter := bson.M{
		"email": email,
	}
	res, err := database.subscriptions.DeleteOne(ctx, filter)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return NotFoundError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	if res.DeletedCount == 0 {
		return NotFoundError(err)
	}
	return nil
}

// InsertOutbox may return the following error codes:
//   - ObjectIDInvalid when the subscription ID is not a valid BSON ObjectID;
//   - CodeNotFound when MongoDB reports that no document was found;
//   - ports.CodeInternal when MongoDB fails to insert the outbox document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) InsertOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, outbox domain.Subscription) error {
	document, err := NewOutboxSubscriptionSchema(outbox, event)
	if err != nil {
		return err
	}
	res, err := database.outbox.InsertOne(ctx, document)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return DuplicateExistsError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	return nil
}

// FindOutbox may return the following error codes:
//   - CodeNotFound when no outbox document matches the email and event;
//   - ports.CodeInternal when MongoDB fails to find or decode the document;
//   - ObjectIDInvalid when the stored outbox ID or AggregateID is empty.
func (database *Database) FindOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email string) (*domain.OutboxSubscription, error) {
	opts := options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})
	filter := bson.M{
		"email": email,
		"event": event,
	}
	var output OutboxSubscriptionSchema
	if err := database.outbox.FindOne(ctx, filter, opts).Decode(&output); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, NotFoundError(err)
		}
		return nil, InternalError(err)
	}
	return output.ToDomain()

}

// RenewConfirmation may return the following error codes:
//   - CodeNotFound when no subscription matches the email;
//   - ports.CodeInternal when MongoDB fails to update the document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) RenewConfirmation(ctx context.Context, email, token string) error {
	filter := bson.M{
		"email": email,
	}
	document := bson.M{
		"$set": bson.M{
			"confirmation_token": token,
			"updated_at":         time.Now(),
		},
		"$unset": bson.M{
			"unsubscribe_token": "",
		},
	}
	res, err := database.subscriptions.UpdateOne(ctx, filter, document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return NotFoundError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	if res.MatchedCount == 0 {
		return NotFoundError(nil)
	}
	return nil
}

// RenewUnsubscribed may return the following error codes:
//   - CodeNotFound when no subscription matches the email;
//   - ports.CodeInternal when MongoDB fails to update the document;
//   - CodeNotAcknowledged when MongoDB does not acknowledge the operation.
func (database *Database) RenewUnsubscribed(ctx context.Context, email, token string) error {
	filter := bson.M{
		"email": email,
	}
	document := bson.M{
		"$set": bson.M{
			"unsubscribe_token": token,
			"updated_at":        time.Now(),
		},
		"$unset": bson.M{
			"confirmation_token": "",
		},
	}
	res, err := database.subscriptions.UpdateOne(ctx, filter, document)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return NotFoundError(err)
		}
		return InternalError(err)
	}
	if !res.Acknowledged {
		return NotAcknowledgedError(nil)
	}
	if res.MatchedCount == 0 {
		return NotFoundError(nil)
	}
	return nil
}

// RenewConfirmationWithOutbox may return the following error codes:
//   - CodeFailedStartSession when MongoDB cannot start a session;
//   - CodeNotFound when the subscription cannot be renewed or found, or the
//     outbox insert reports no document;
//   - ports.CodeInternal when a subscription or outbox operation fails;
//   - CodeNotAcknowledged when MongoDB does not acknowledge an update or insert;
//   - ObjectIDInvalid when a stored subscription ID is empty or cannot be
//     converted to a BSON ObjectID.
//
// WithTransaction may also return a MongoDB error without a ports.Code.
func (database *Database) RenewConfirmationWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, confirmation string) error {
	session, err := database.subscriptions.Database().Client().StartSession()
	if err != nil {
		return FailedStartSessionError(err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tctx context.Context) (any, error) {
		if err := database.RenewConfirmation(tctx, email, confirmation); err != nil {
			return nil, err
		}
		subscription, err := database.Find(tctx, email)
		if err != nil {
			return nil, err
		}
		if err := database.InsertOutbox(tctx, event, *subscription); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// RenewUnsubscribedWithOutbox may return the following error codes:
//   - CodeFailedStartSession when MongoDB cannot start a session;
//   - CodeNotFound when the subscription cannot be renewed or found, or the
//     outbox insert reports no document;
//   - ports.CodeInternal when a subscription or outbox operation fails;
//   - CodeNotAcknowledged when MongoDB does not acknowledge an update or insert;
//   - ObjectIDInvalid when a stored subscription ID is empty or cannot be
//     converted to a BSON ObjectID.
//
// WithTransaction may also return a MongoDB error without a ports.Code.
func (database *Database) RenewUnsubscribedWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, unsubscribe string) error {
	session, err := database.subscriptions.Database().Client().StartSession()
	if err != nil {
		return FailedStartSessionError(err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tctx context.Context) (any, error) {
		if err := database.RenewUnsubscribed(tctx, email, unsubscribe); err != nil {
			return nil, err
		}
		subscription, err := database.Find(tctx, email)
		if err != nil {
			return nil, err
		}
		if err := database.InsertOutbox(tctx, event, *subscription); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}

// InsertWithOutbox may return the following error codes:
//   - CodeFailedStartSession when MongoDB cannot start a session;
//   - ObjectIDInvalid when the subscription ID is not a valid BSON ObjectID;
//   - CodeDuplicateExists when a subscription with the same unique key exists;
//   - CodeNotFound when the outbox insert reports no document;
//   - ports.CodeInternal when a subscription or outbox insert fails;
//   - CodeNotAcknowledged when MongoDB does not acknowledge an insert.
//
// WithTransaction may also return a MongoDB error without a ports.Code.
func (database *Database) InsertWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, subscription domain.Subscription) error {
	session, err := database.subscriptions.Database().Client().StartSession()
	if err != nil {
		return FailedStartSessionError(err)
	}
	defer session.EndSession(ctx)
	_, err = session.WithTransaction(ctx, func(tctx context.Context) (any, error) {
		if err := database.Insert(tctx, subscription); err != nil {
			return nil, err
		}
		if err := database.InsertOutbox(tctx, event, subscription); err != nil {
			return nil, err
		}
		return nil, nil
	})
	return err
}
