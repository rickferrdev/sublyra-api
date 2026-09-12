package env

import (
	"github.com/rickferrdev/dotenv"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

type Env struct {
	ServerPort          string `env:"SERVER_PORT" default:"8080"`
	ServerHost          string `env:"SERVER_HOST" default:"localhost"`
	AppScheme           string `env:"APP_SCHEME" default:"http"`
	AppHost             string `env:"APP_HOST" default:"localhost:8080"`
	MongoURI            string `env:"MONGO_URI" required:"true"`
	ResendSecretKey     string `env:"RESEND_SECRET_KEY" required:"true"`
	JwtSecretKey        string `env:"JWT_SECRET_KEY" required:"true"`
	RabbitMQURI         string `env:"RABBITMQ_URI" required:"true"`
	OutboxPollInterval  string `env:"OUTBOX_POLL_INTERVAL" default:"2s"`
	OutboxBatchSize     string `env:"OUTBOX_BATCH_SIZE" default:"50"`
	OutboxMaxAttempts   string `env:"OUTBOX_MAX_ATTEMPTS" default:"5"`
	ResendFromEmail     string `env:"RESEND_FROM_EMAIL" default:"onboarding@resend.dev"`
	RabbitMQPrefetch    string `env:"RABBITMQ_PREFETCH" default:"5"`
	OutboxTTLSeconds    string `env:"OUTBOX_TTL_SECONDS" default:"604800"`
	CorsFrontendAllowed string `env:"CORS_FRONTEND_ALLOWED" default:"http://localhost:3000"`
	HealthTimeout       string `env:"HEALTH_TIMEOUT" default:"5xxx"`
}

func New() (*Env, error) {
	var env Env
	dotenv.Collect()

	if err := dotenv.Unmarshal(&env); err != nil {
		return nil, err
	}

	return &env, nil
}
