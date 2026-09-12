package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rickferrdev/sublyra-api/internal/core/domain"
	"github.com/rickferrdev/sublyra-api/internal/core/ports"
	"github.com/rickferrdev/sublyra-api/internal/inbound/http/rest/constants"
	dbsubscription "github.com/rickferrdev/sublyra-api/internal/outbound/mongodb/repositories/subscription"
	"github.com/rickferrdev/sublyra-api/internal/platform/jwttoken"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/fx"
)

var Provide = fx.Provide(fx.Annotate(New, fx.As(new(Interface))))

type Service struct {
	jwttoken jwttoken.Interface
	database dbsubscription.Interface
}
type Interface interface {
	RegisterSubscription(ctx context.Context, email, name string) error
	RegisterUnsubscription(ctx context.Context, email string) error

	SubscriptionConfirm(ctx context.Context) error
	UnsubscriptionConfirm(ctx context.Context) error
}

type FxParams struct {
	fx.In
	JwtToken jwttoken.Interface
	Database dbsubscription.Interface
}

func New(params FxParams) *Service {
	service := Service{
		jwttoken: params.JwtToken,
		database: params.Database,
	}

	return &service
}

func (service *Service) RegisterSubscription(ctx context.Context, email, name string) error {
	subscription, err := service.database.Find(ctx, email)
	if err != nil {
		if subscription == nil && ports.IsCode(err, dbsubscription.CodeNotFound) {
			return service.CreateSubscription(ctx, email, name)
		}
		return ports.Internal(err)
	}
	if !subscription.IsUnsubscribed() && !subscription.IsPending() {
		return ports.Conflict(errors.New("existing subscription must have a status of \"pending\" or \"unsubscribed\""))
	}
	if err := subscription.CooldownUnsubscription(); err != nil {
		return err
	}
	return service.RenewConfirmation(ctx, email)
}

func (service *Service) RegisterUnsubscription(ctx context.Context, email string) error {
	subscription, err := service.database.Find(ctx, email)
	if err != nil {
		if ports.IsCode(err, dbsubscription.CodeNotFound) {
			return ports.Unauthorized(err)
		}
		return ports.Internal(err)
	}
	if !subscription.IsSubscribed() {
		return ports.Unauthorized(errors.New("cannot cancel a subscription if you are not subscribed"))
	}
	return service.RenewUnsubscribed(ctx, email)
}

func (service *Service) SubscriptionConfirm(ctx context.Context) error {
	auth, ok := ctx.Value(constants.SUBSCRIPTION_AUTH_KEY).(constants.SubscriptionAuth)
	if !ok {
		return ports.Unauthorized(errors.New("expected auth token"))
	}
	email := auth.Claims.Data
	token := auth.Token
	if !ok {
		return ports.Unauthorized(errors.New("expected valid token"))
	}
	subscription, err := service.database.Find(ctx, email)
	if err != nil {
		if ports.IsCode(err, dbsubscription.CodeNotFound) {
			return ports.Unauthorized(err)
		}
		return ports.Internal(err)
	}
	if subscription == nil {
		return ports.Unauthorized(errors.New("corrupted subscription"))
	}
	if !subscription.CompareTokenConfirmation(token) {
		return ports.Unauthorized(errors.New("invalid tokens"))
	}
	if !subscription.IsPending() && !subscription.IsUnsubscribed() {
		return ports.Unauthorized(errors.New("invalid subscription status"))
	}
	var expiresAt time.Time
	switch subscription.Name {
	case domain.SubscriptionNameBasic:
	case domain.SubscriptionNamePro, domain.SubscriptionNameBusiness:
		expiresAt = time.Now().AddDate(0, 1, 0)
	default:
		return ports.Internal(errors.New("subscription name is corrupt"))
	}
	status := domain.SubscriptionStatusSubscribed
	subscribedAt, updatedAt := time.Now(), time.Now()
	if err := service.database.Update(ctx, email, dbsubscription.SubscriptionUpdateSchema{
		Status:       &status,
		SubscribedAt: &subscribedAt,
		ExpiresAt:    &expiresAt,
		UpdatedAt:    &updatedAt,
		Unset:        []string{"unsubscribed_at", "confirmation_token"},
	}); err != nil {
		if ports.IsCode(err, dbsubscription.CodeNotFound) {
			return ports.Unauthorized(err)
		}
		return ports.Internal(err)
	}
	return nil
}

func (service *Service) UnsubscriptionConfirm(ctx context.Context) error {
	auth, ok := ctx.Value(constants.SUBSCRIPTION_AUTH_KEY).(constants.SubscriptionAuth)
	if !ok {
		return ports.Unauthorized(errors.New("expected auth token"))
	}
	email := auth.Claims.Data
	token := auth.Token
	subscription, err := service.database.Find(ctx, email)
	if err != nil {
		if ports.IsCode(err, dbsubscription.CodeNotFound) {
			return ports.Unauthorized(err)
		}
		return ports.Internal(err)
	}
	if subscription == nil {
		return ports.Unauthorized(errors.New("corrupted subscription"))
	}
	if !subscription.CompareTokenUnsubscribe(token) {
		return ports.Unauthorized(errors.New("invalid tokens"))
	}
	if !subscription.IsSubscribed() {
		return ports.Unauthorized(errors.New("subscription status must be subscribed to proceed"))
	}
	status := domain.SubscriptionStatusUnsubscribed
	unsubscribedAt, updatedAt := time.Now(), time.Now()
	if err := service.database.Update(ctx, email, dbsubscription.SubscriptionUpdateSchema{
		Status:         &status,
		UnsubscribedAt: &unsubscribedAt,
		UpdatedAt:      &updatedAt,
		Unset:          []string{"unsubscribe_token", "subscribed_at"},
	}); err != nil {
		if ports.IsCode(err, dbsubscription.CodeNotFound) {
			return ports.Unauthorized(err)
		}
		return ports.Internal(err)
	}
	return nil
}

func (service *Service) GenerateToken(email string, subject string) (string, error) {
	token, err := service.jwttoken.GenerateToken(jwttoken.Claims{
		Data: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    jwttoken.Issuer,
			Audience:  []string{jwttoken.Audience},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	})
	if err != nil {
		return "", ports.Internal(err)
	}
	return token, nil
}

func (service *Service) CreateSubscription(ctx context.Context, email, name string) error {
	token, err := service.GenerateToken(email, string(domain.SubscriptionStatusSubscribed))
	if err != nil {
		return err
	}
	var (
		subscriptionName  domain.SubscriptionName
		subscriptionPrice domain.SubscriptionPrice
	)
	switch name {
	case string(domain.SubscriptionNameBasic):
		subscriptionName = domain.SubscriptionNameBasic
		subscriptionPrice = domain.SubscriptionPriceBasic
	case string(domain.SubscriptionNamePro):
		subscriptionName = domain.SubscriptionNamePro
		subscriptionPrice = domain.SubscriptionPricePro
	case string(domain.SubscriptionNameBusiness):
		subscriptionName = domain.SubscriptionNameBusiness
		subscriptionPrice = domain.SubscriptionPriceBusiness
	}
	subscription := domain.Subscription{
		ID:                bson.NewObjectID().Hex(),
		Name:              subscriptionName,
		Price:             subscriptionPrice,
		Email:             email,
		Status:            domain.SubscriptionStatusPending,
		ConfirmationToken: token,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	// isso com o price, expires at e name...
	if err := service.database.InsertWithOutbox(
		ctx,
		domain.EventOutboxSubscriptionConfirmationRequested,
		subscription,
	); err != nil {
		if ports.IsCode(err, dbsubscription.CodeDuplicateExists) {
			return ports.Conflict(err)
		}
		return ports.Internal(err)
	}
	return nil
}

func (service *Service) RenewConfirmation(ctx context.Context, email string) error {
	token, err := service.GenerateToken(email, string(domain.SubscriptionStatusSubscribed))
	if err != nil {
		return err
	}
	if err := service.database.RenewConfirmationWithOutbox(
		ctx,
		domain.EventOutboxSubscriptionConfirmationRequested,
		email,
		token,
	); err != nil {
		return ports.Internal(err)
	}
	return nil
}

func (service *Service) RenewUnsubscribed(ctx context.Context, email string) error {
	token, err := service.GenerateToken(email, string(domain.SubscriptionStatusUnsubscribed))
	if err != nil {
		return err
	}
	if err := service.database.RenewUnsubscribedWithOutbox(
		ctx,
		domain.EventOutboxSubscriptionCancellationRequested,
		email,
		token,
	); err != nil {
		return ports.Internal(err)
	}
	return nil
}
