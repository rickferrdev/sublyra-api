package validator

import (
	"github.com/go-playground/validator/v10"
	"go.uber.org/fx"
)

var Provide = fx.Provide(fx.Annotate(New, fx.As(new(Interface))))

type Validator struct {
	validate *validator.Validate
}

type Interface interface {
	Validate(out any) error
}

func New() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

func (validator *Validator) Validate(out any) error {
	return validator.validate.Struct(out)
}
