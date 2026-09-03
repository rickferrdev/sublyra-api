package jwttoken

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
	"go.uber.org/fx"
)

const (
	Issuer   = "sublyra-api"
	Audience = "subscriptions"
)

var Provide = fx.Provide(
	fx.Annotate(
		New,
		fx.As(new(Interface)),
	),
)

type Platform struct {
	secret []byte
}

type Params struct {
	fx.In

	Env *env.Env
}

type Interface interface {
	GenerateToken(claims Claims) (string, error)
	ValidateToken(token string) (*Claims, error)
}

func New(params Params) Interface {
	return &Platform{
		secret: []byte(params.Env.JwtSecretKey),
	}
}

type Claims struct {
	Data string `json:"data"`
	jwt.RegisteredClaims
}

func (platform *Platform) GenerateToken(claims Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		Data:             claims.Data,
		RegisteredClaims: claims.RegisteredClaims,
	})

	return token.SignedString(platform.secret)

}

func (platform *Platform) ValidateToken(token string) (*Claims, error) {
	pretoken, err := jwt.ParseWithClaims(token, &Claims{}, func(token *jwt.Token) (any, error) {
		return platform.secret, nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(Issuer),
		jwt.WithAudience(Audience),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
	)
	if err != nil {
		return nil, err
	}

	claims, ok := pretoken.Claims.(*Claims)
	if !ok || !pretoken.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}

	return claims, nil
}
