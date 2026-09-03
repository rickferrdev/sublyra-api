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
			models := []mongo.IndexModel{
				{Keys: bson.M{"email": 1}, Options: options.Index().SetName("subscriptions_email_unique").SetUnique(true)},
			}
			if _, err := subscription.Indexes().CreateMany(ctx, models); err != nil {
				return err
			}
			return nil
		},
		OnStop: context.Cause,
	})
}
