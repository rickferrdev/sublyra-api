package mailer

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"html/template"

	"github.com/resend/resend-go/v4"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"go.uber.org/fx"
)

var Provide = fx.Provide(fx.Annotate(New, fx.As(new(Interface))))

type Mailer struct {
	env    *env.Env
	client *resend.Client
}

type Interface interface {
	Send(ctx context.Context, subject string, recipient []string, email Email) error
	EmailConfirmation(data ConfirmationData) Email
	EmailCancellation(data CancellationData) Email
}

type FxParams struct {
	fx.In
	Env *env.Env
}

func New(params FxParams) *Mailer {
	mailer := Mailer{
		env:    params.Env,
		client: resend.NewClient(params.Env.ResendSecretKey),
	}

	return &mailer
}

type (
	CancellationData struct {
		CancellationURL string
		Email           string
	}
	ConfirmationData struct {
		ConfirmationURL string
		Email           string
	}
)

//go:embed emails/*.html
var templateFS embed.FS
var templates = template.Must(
	template.ParseFS(templateFS, "emails/*.html"),
)

type (
	Email func() (string, error)
)

func (mailer *Mailer) Send(ctx context.Context, subject string, recipients []string, email Email) error {
	if subject == "" {
		return ports.Internal(errors.New("invalid subject"))
	}

	if len(recipients) == 0 {
		return ports.Internal(errors.New("invalid recipients"))
	}

	bytes, err := email()
	if err != nil {
		return ports.Internal(err)
	}
	_, err = mailer.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From: mailer.env.ResendFromEmail,
		To:      recipients,
		Subject: subject,
		Html:    bytes,
	})
	return err
}

func (mailer *Mailer) EmailConfirmation(data ConfirmationData) Email {
	return mailer.render("confirmation.html", data)
}

func (mailer *Mailer) EmailCancellation(data CancellationData) Email {
	return mailer.render("cancellation.html", data)
}

func (mailer *Mailer) render(name string, data any) Email {
	return func() (string, error) {
		var bytes bytes.Buffer
		err := templates.ExecuteTemplate(&bytes, name, data)
		return bytes.String(), err
	}
}
