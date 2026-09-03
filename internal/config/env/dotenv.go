package env

import (
	"github.com/rickferrdev/dotenv"
	"go.uber.org/fx"
)

var Provide = fx.Provide(New)

type Env struct {
	ServerPort string `env:"SERVER_PORT" default:"8080"`
	ServerHost string `env:"SERVER_HOST" default:"localhost"`

	MongoURI string `env:"MONGO_URI" required:"true"`

	ResendSecretKey string `env:"RESEND_SECRET_KEY" required:"true"`
	JwtSecretKey    string `env:"JWT_SECRET_KEY" required:"true"`
}

func New() (*Env, error) {
	var env Env
	dotenv.Collect()

	if err := dotenv.Unmarshal(&env); err != nil {
		return nil, err
	}

	return &env, nil
}
