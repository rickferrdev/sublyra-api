package subscription

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/fx"
)

var Invoke = fx.Invoke(StartIndexes)

type FxIndexesParams struct {
	fx.In
	Lifecycle fx.Lifecycle
	Client    *mongo.Client
}

func StartIndexes(params FxIndexesParams) {
	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			subscription := params.Client.Database(SUBSCRIPTIONS_DATABASE).Collection(SUBSCRIPTIONS_COLLECTION)
			outbox := params.Client.Database(SUBSCRIPTIONS_DATABASE).Collection(OUTBOX_COLLECTION)
			subscriptionModels := []mongo.IndexModel{
				{Keys: bson.M{"email": 1}, Options: options.Index().SetName("subscriptions_email_unique").SetUnique(true)},
			}
			outboxModels := []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "status", Value: 1},
						{Key: "attempts", Value: 1},
						{Key: "created_at", Value: 1},
					},
					Options: options.Index().SetName("outbox_pending_attempts_created_at"),
				},
			}
			if _, err := subscription.Indexes().CreateMany(ctx, subscriptionModels); err != nil {
				return err
			}
			if _, err := outbox.Indexes().CreateMany(ctx, outboxModels); err != nil {
				return err
			}
			return nil
		},
		OnStop: context.Cause,
	})
}
