package mongo

import (
	"context"

	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Env       *env.Env
}

func New(params Params) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(params.Env.MongoURI))
	if err != nil {
		return nil, err
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return client.Ping(ctx, readpref.PrimaryPreferred())
		},
		OnStop: func(ctx context.Context) error {
			return client.Disconnect(ctx)
		},
	})

	return client, nil
}
