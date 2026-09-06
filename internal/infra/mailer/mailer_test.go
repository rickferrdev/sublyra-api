package mailer_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/infra/mailer"
)

func TestMailer_EmailConfirmation(t *testing.T) {
	m := mailer.New(mailer.FxParams{
		Env: &env.Env{
			ResendSecretKey: "re_test_123",
			ResendFromEmail: "onboarding@resend.dev",
		},
	})

	emailFn := m.EmailConfirmation(mailer.ConfirmationData{
		Recipients:      []string{"user@example.com"},
		Subject:         "Confirm your subscription",
		ConfirmationURL: "https://api.examples.com/subscription/confirm?token=abc",
		Email:           "user@example.com",
	})

	req, err := emailFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Subject != "Confirm your subscription" {
		t.Errorf("expected subject 'Confirm your subscription', got %q", req.Subject)
	}
	if len(req.Recipients) != 1 || req.Recipients[0] != "user@example.com" {
		t.Errorf("expected recipient 'user@example.com', got %v", req.Recipients)
	}
	if !strings.Contains(req.BytesHTML, "https://api.examples.com/subscription/confirm?token=abc") {
		t.Errorf("expected HTML to contain confirmation URL, got: %s", req.BytesHTML)
	}
}

func TestMailer_EmailCancellation(t *testing.T) {
	m := mailer.New(mailer.FxParams{
		Env: &env.Env{
			ResendSecretKey: "re_test_123",
			ResendFromEmail: "onboarding@resend.dev",
		},
	})

	emailFn := m.EmailCancellation(mailer.CancellationData{
		Recipients:      []string{"user@example.com"},
		Subject:         "Cancel your subscription",
		CancellationURL: "https://api.examples.com/unsubscription/confirm?token=xyz",
		Email:           "user@example.com",
	})

	req, err := emailFn()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if req.Subject != "Cancel your subscription" {
		t.Errorf("expected subject 'Cancel your subscription', got %q", req.Subject)
	}
	if !strings.Contains(req.BytesHTML, "https://api.examples.com/unsubscription/confirm?token=xyz") {
		t.Errorf("expected HTML to contain cancellation URL, got: %s", req.BytesHTML)
	}
}

func TestMailer_ValidationErrors(t *testing.T) {
	m := mailer.New(mailer.FxParams{
		Env: &env.Env{
			ResendSecretKey: "re_test_123",
		},
	})

	t.Run("missing subject", func(t *testing.T) {
		emailFn := m.EmailConfirmation(mailer.ConfirmationData{
			Recipients: []string{"user@example.com"},
			Subject:    "",
		})
		_, err := emailFn()
		if err == nil {
			t.Error("expected error for missing subject, got nil")
		}
	})

	t.Run("missing recipients", func(t *testing.T) {
		emailFn := m.EmailConfirmation(mailer.ConfirmationData{
			Recipients: []string{},
			Subject:    "Test",
		})
		_, err := emailFn()
		if err == nil {
			t.Error("expected error for missing recipients, got nil")
		}
	})

	t.Run("send with invalid function payload", func(t *testing.T) {
		ctx := context.Background()
		err := m.Send(ctx, func() (*mailer.EmailRequest, error) {
			return nil, errors.New("simulated error")
		})
		if err == nil {
			t.Error("expected error from Send when function fails, got nil")
		}
	})
}
