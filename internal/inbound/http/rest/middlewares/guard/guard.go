package guard

import "go.uber.org/fx"

var Invoke = fx.Invoke(New)

type Middleware struct{}

func New() (*Middleware, error) {
	return &Middleware{}, nil
}

func (handler *Middleware) GuardToken() error {
	return nil
}
