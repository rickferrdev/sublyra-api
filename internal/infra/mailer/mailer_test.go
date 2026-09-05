package mailer

import (
	"strings"
	"testing"
)

func TestEmailTemplatesRenderRecipientAndActionURL(t *testing.T) {
	mailer := &Mailer{}
	tests := []struct {
		name  string
		email Email
		want  []string
	}{
		{"confirmation", mailer.EmailConfirmation(ConfirmationData{Email: "person@example.com", ConfirmationURL: "https://example.com/confirm?token=test"}), []string{"person@example.com", "https://example.com/confirm?token=test", "CONFIRM SUBSCRIPTION"}},
		{"cancellation", mailer.EmailCancellation(CancellationData{Email: "person@example.com", CancellationURL: "https://example.com/cancel?token=test"}), []string{"person@example.com", "https://example.com/cancel?token=test", "CONFIRM UNSUBSCRIPTION"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			content, err := test.email()
			if err != nil {
				t.Fatalf("render template: %v", err)
			}
			for _, expected := range test.want {
				if !strings.Contains(content, expected) {
					t.Fatalf("rendered email does not contain %q", expected)
				}
			}
		})
	}
}
