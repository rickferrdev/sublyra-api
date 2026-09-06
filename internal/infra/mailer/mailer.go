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
	Send(ctx context.Context, email Email) error
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
		Recipients      []string
		Subject         string
		CancellationURL string
		Email           string
	}
	ConfirmationData struct {
		Recipients      []string
		Subject         string
		ConfirmationURL string
		Email           string
	}
)

type (
	EmailRequest struct {
		Recipients []string
		Subject    string
		BytesHTML  string
	}

	RenderRequest struct {
		Recipients   []string
		Subject      string
		TemplateName string
		Template     any
	}
)

//go:embed emails/*.html
var templateFS embed.FS
var templates = template.Must(
	template.ParseFS(templateFS, "emails/*.html"),
)

type (
	Email func() (*EmailRequest, error)
)

func (mailer *Mailer) Send(ctx context.Context, email Email) error {
	data, err := email()
	if err != nil {
		return ports.Internal(err)
	}
	_, err = mailer.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    mailer.env.ResendFromEmail,
		To:      data.Recipients,
		Subject: data.Subject,
		Html:    data.BytesHTML,
	})
	return err
}

func (mailer *Mailer) EmailConfirmation(data ConfirmationData) Email {
	return mailer.render(RenderRequest{
		TemplateName: "confirmation.html",
		Recipients:   data.Recipients,
		Template:     data,
		Subject:      data.Subject,
	})
}

func (mailer *Mailer) EmailCancellation(data CancellationData) Email {
	return mailer.render(RenderRequest{
		TemplateName: "cancellation.html",
		Recipients:   data.Recipients,
		Template:     data,
		Subject:      data.Subject,
	})
}

func (mailer *Mailer) render(data RenderRequest) Email {
	return func() (*EmailRequest, error) {
		if data.Subject == "" {
			return nil, ports.Internal(errors.New("invalid subject"))
		}
		if len(data.Recipients) == 0 {
			return nil, ports.Internal(errors.New("invalid recipients"))
		}
		var bytes bytes.Buffer
		if err := templates.ExecuteTemplate(&bytes, data.TemplateName, data.Template); err != nil {
			return nil, err
		}
		return &EmailRequest{
			Recipients: data.Recipients,
			Subject:    data.Subject,
			BytesHTML:  bytes.String(),
		}, nil
	}
}
