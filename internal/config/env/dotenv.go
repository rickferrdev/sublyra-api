package env

import (
	"github.com/rickferrdev/dotenv"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

type Env struct {
	ServerPort         string `env:"SERVER_PORT" default:"8080"`
	ServerHost         string `env:"SERVER_HOST" default:"localhost"`
	MongoURI           string `env:"MONGO_URI" required:"true"`
	ResendSecretKey    string `env:"RESEND_SECRET_KEY" required:"true"`
	JwtSecretKey       string `env:"JWT_SECRET_KEY" required:"true"`
	RabbitMQURI        string `env:"RABBITMQ_URI" required:"true"`
	OutboxPollInterval string `env:"OUTBOX_POLL_INTERVAL" default:"2s"`
	OutboxBatchSize    int    `env:"OUTBOX_BATCH_SIZE" default:"50"`
	OutboxMaxAttempts  int    `env:"OUTBOX_MAX_ATTEMPTS" default:"5"`
	ResendFromEmail    string `env:"RESEND_FROM_EMAIL" default:"onboarding@resend.dev"`
	RabbitMQPrefetch   string `env:"RABBITMQ_PREFETCH" default:"5"`
}

func New() (*Env, error) {
	var env Env
	dotenv.Collect()

	if err := dotenv.Unmarshal(&env); err != nil {
		return nil, err
	}

	return &env, nil
}
