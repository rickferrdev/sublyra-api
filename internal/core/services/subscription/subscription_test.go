package subscription_test

import (
	"context"
	"testing"
	"time"

	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/core/services/subscription"
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/constants"
	dbsubscription "github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"github.com/rickferrdev/sublyra-api/internal/platform/jwttoken"
)

type mockDatabase struct {
	subscriptions map[string]*domain.Subscription
}

func newMockDatabase() *mockDatabase {
	return &mockDatabase{
		subscriptions: make(map[string]*domain.Subscription),
	}
}

func (m *mockDatabase) Insert(ctx context.Context, sub domain.Subscription) error {
	m.subscriptions[sub.Email] = &sub
	return nil
}

func (m *mockDatabase) Find(ctx context.Context, email string) (*domain.Subscription, error) {
	sub, ok := m.subscriptions[email]
	if !ok {
		return nil, dbsubscription.NotFoundError(dbsubscription.ErrNotFound)
	}
	return sub, nil
}

func (m *mockDatabase) Update(ctx context.Context, email string, update dbsubscription.SubscriptionUpdateSchema) error {
	sub, ok := m.subscriptions[email]
	if !ok {
		return dbsubscription.NotFoundError(dbsubscription.ErrNotFound)
	}
	if update.Status != nil {
		sub.Status = *update.Status
	}
	if update.SubscribedAt != nil {
		sub.SubscribedAt = *update.SubscribedAt
	}
	if update.UnsubscribedAt != nil {
		sub.UnsubscribedAt = *update.UnsubscribedAt
	}
	for _, unset := range update.Unset {
		switch unset {
		case "confirmation_token":
			sub.ConfirmationToken = ""
		case "unsubscribe_token":
			sub.UnsubscribeToken = ""
		case "unsubscribed_at":
			sub.UnsubscribedAt = time.Time{}
		case "subscribed_at":
			sub.SubscribedAt = time.Time{}
		}
	}
	return nil
}

func (m *mockDatabase) Delete(ctx context.Context, email string) error {
	delete(m.subscriptions, email)
	return nil
}

func (m *mockDatabase) RenewConfirmation(ctx context.Context, email, token string) error {
	if sub, ok := m.subscriptions[email]; ok {
		sub.ConfirmationToken = token
	}
	return nil
}

func (m *mockDatabase) RenewUnsubscribed(ctx context.Context, email, token string) error {
	if sub, ok := m.subscriptions[email]; ok {
		sub.UnsubscribeToken = token
	}
	return nil
}

func (m *mockDatabase) InsertOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, outbox domain.Subscription) error {
	return nil
}

func (m *mockDatabase) FindOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email string) (*domain.OutboxSubscription, error) {
	return nil, nil
}

func (m *mockDatabase) InsertWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, sub domain.Subscription) error {
	m.subscriptions[sub.Email] = &sub
	return nil
}

func (m *mockDatabase) RenewConfirmationWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, token string) error {
	if sub, ok := m.subscriptions[email]; ok {
		sub.ConfirmationToken = token
	}
	return nil
}

func (m *mockDatabase) RenewUnsubscribedWithOutbox(ctx context.Context, event domain.OutboxSubscriptionEvent, email, token string) error {
	if sub, ok := m.subscriptions[email]; ok {
		sub.UnsubscribeToken = token
	}
	return nil
}

func (m *mockDatabase) BatchFindOutboxByStatus(ctx context.Context, status domain.OutboxSubscriptionStatus, limit int) ([]domain.OutboxSubscription, error) {
	return nil, nil
}

func (m *mockDatabase) MarkPublished(ctx context.Context, id string, publishedAt time.Time) error {
	return nil
}

func (m *mockDatabase) MarkFailed(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockDatabase) MarkProcessing(ctx context.Context, id string) error {
	return nil
}

func (m *mockDatabase) MarkDelivered(ctx context.Context, id string) error {
	return nil
}

func (m *mockDatabase) IncrementAttempts(ctx context.Context, id string, reason string) error {
	return nil
}

func (m *mockDatabase) ClaimPendingOutbox(ctx context.Context, maxAttempts int) (*domain.OutboxSubscription, error) {
	return nil, nil
}

func setupService() (*subscription.Service, *mockDatabase, jwttoken.Interface) {
	db := newMockDatabase()
	jwtPlatform := jwttoken.New(jwttoken.Params{
		Env: &env.Env{JwtSecretKey: "supersecretkeyforunitest1234567890"},
	})
	svc := subscription.New(subscription.FxParams{
		JwtToken: jwtPlatform,
		Database: db,
	})
	return svc, db, jwtPlatform
}

func TestRegisterSubscription(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := setupService()

	t.Run("new email registration", func(t *testing.T) {
		err := svc.RegisterSubscription(ctx, "new@example.com", "blu")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		sub, err := db.Find(ctx, "new@example.com")
		if err != nil {
			t.Fatalf("expected subscription created, got error %v", err)
		}
		if sub.Status != domain.SubscriptionStatusPending {
			t.Errorf("expected pending status, got %s", sub.Status)
		}
		if sub.ConfirmationToken == "" {
			t.Errorf("expected non-empty confirmation token")
		}
	})

	t.Run("conflict when already subscribed", func(t *testing.T) {
		db.subscriptions["active@example.com"] = &domain.Subscription{
			Email:  "active@example.com",
			Status: domain.SubscriptionStatusSubscribed,
		}
		err := svc.RegisterSubscription(ctx, "active@example.com", "blu")
		if err == nil {
			t.Fatal("expected conflict error, got nil")
		}
		if !ports.IsCode(err, ports.CodeConflict) {
			t.Errorf("expected conflict status code, got %v", err)
		}
	})
}

func TestRegisterUnsubscription(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := setupService()

	t.Run("not subscribed user cannot unsubscribe", func(t *testing.T) {
		err := svc.RegisterUnsubscription(ctx, "nonexistent@example.com")
		if err == nil {
			t.Fatal("expected unauthorized error for non-existent user, got nil")
		}
		if !ports.IsCode(err, ports.CodeUnauthorized) {
			t.Errorf("expected unauthorized code, got %v", err)
		}
	})

	t.Run("subscribed user can request unsubscription", func(t *testing.T) {
		db.subscriptions["subscribed@example.com"] = &domain.Subscription{
			Email:  "subscribed@example.com",
			Status: domain.SubscriptionStatusSubscribed,
		}
		err := svc.RegisterUnsubscription(ctx, "subscribed@example.com")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		sub := db.subscriptions["subscribed@example.com"]
		if sub.UnsubscribeToken == "" {
			t.Errorf("expected unsubscribe token to be generated")
		}
	})
}

func TestSubscriptionConfirm(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := setupService()

	t.Run("valid token confirms subscription", func(t *testing.T) {
		email := "confirm@example.com"
		token, err := svc.GenerateToken(email, string(domain.SubscriptionStatusSubscribed))
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		db.subscriptions[email] = &domain.Subscription{
			Email:             email,
			Status:            domain.SubscriptionStatusPending,
			ConfirmationToken: token,
		}

		auth := constants.SubscriptionAuth{
			Claims: &jwttoken.Claims{Data: email},
			Token:  token,
		}
		testCtx := context.WithValue(ctx, constants.SUBSCRIPTION_AUTH_KEY, auth)

		err = svc.SubscriptionConfirm(testCtx)
		if err != nil {
			t.Fatalf("expected confirmation success, got %v", err)
		}

		sub := db.subscriptions[email]
		if sub.Status != domain.SubscriptionStatusSubscribed {
			t.Errorf("expected subscribed status, got %s", sub.Status)
		}
		if sub.ConfirmationToken != "" {
			t.Errorf("expected confirmation token to be cleared")
		}
	})

	t.Run("invalid token fails", func(t *testing.T) {
		err := svc.SubscriptionConfirm(ctx)
		if err == nil {
			t.Fatal("expected unauthorized error for missing auth token in context, got nil")
		}
	})

	t.Run("mismatched token fails", func(t *testing.T) {
		email := "mismatch@example.com"
		token1, _ := svc.GenerateToken(email, string(domain.SubscriptionStatusSubscribed))
		token2, _ := svc.GenerateToken(email, "different_subject")

		db.subscriptions[email] = &domain.Subscription{
			Email:             email,
			Status:            domain.SubscriptionStatusPending,
			ConfirmationToken: token1,
		}

		auth := constants.SubscriptionAuth{
			Claims: &jwttoken.Claims{Data: email},
			Token:  token2,
		}
		testCtx := context.WithValue(ctx, constants.SUBSCRIPTION_AUTH_KEY, auth)

		err := svc.SubscriptionConfirm(testCtx)
		if err == nil {
			t.Fatal("expected error for token mismatch, got nil")
		}
		if !ports.IsCode(err, ports.CodeUnauthorized) {
			t.Errorf("expected unauthorized code, got %v", err)
		}
	})
}

func TestUnsubscriptionConfirm(t *testing.T) {
	ctx := context.Background()
	svc, db, _ := setupService()

	t.Run("valid token confirms unsubscription", func(t *testing.T) {
		email := "unsub@example.com"
		token, err := svc.GenerateToken(email, string(domain.SubscriptionStatusUnsubscribed))
		if err != nil {
			t.Fatalf("failed to generate token: %v", err)
		}
		db.subscriptions[email] = &domain.Subscription{
			Email:            email,
			Status:           domain.SubscriptionStatusSubscribed,
			UnsubscribeToken: token,
		}

		auth := constants.SubscriptionAuth{
			Claims: &jwttoken.Claims{Data: email},
			Token:  token,
		}
		testCtx := context.WithValue(ctx, constants.SUBSCRIPTION_AUTH_KEY, auth)

		err = svc.UnsubscriptionConfirm(testCtx)
		if err != nil {
			t.Fatalf("expected unsubscription confirmation success, got %v", err)
		}

		sub := db.subscriptions[email]
		if sub.Status != domain.SubscriptionStatusUnsubscribed {
			t.Errorf("expected unsubscribed status, got %s", sub.Status)
		}
	})
}

func TestDomain_Errors(t *testing.T) {
	t.Run("cooldown error on rapid resubscribe", func(t *testing.T) {
		sub := &domain.Subscription{
			Status:         domain.SubscriptionStatusUnsubscribed,
			UnsubscribedAt: time.Now(),
		}
		err := sub.CooldownUnsubscription()
		if err == nil {
			t.Error("expected cooldown error when unsubscribed just now")
		}
	})
}
