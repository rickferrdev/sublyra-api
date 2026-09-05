package jwttoken

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rickferrdev/sublyra-api/internal/config/env"
)

func TestGenerateAndValidateToken(t *testing.T) {
	platform := New(Params{Env: &env.Env{JwtSecretKey: "test-secret"}})
	claims := Claims{
		Data: "person@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    Issuer,
			Audience:  []string{Audience},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}

	token, err := platform.GenerateToken(claims)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	validated, err := platform.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if validated.Data != claims.Data {
		t.Fatalf("expected data %q, got %q", claims.Data, validated.Data)
	}
}

func TestValidateTokenRejectsInvalidTokens(t *testing.T) {
	valid := New(Params{Env: &env.Env{JwtSecretKey: "test-secret"}})
	other := New(Params{Env: &env.Env{JwtSecretKey: "other-secret"}})

	tests := []struct {
		name   string
		claims Claims
		issuer Interface
	}{
		{
			name: "expired token",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer: Issuer, Audience: []string{Audience},
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Minute)),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
			}},
			issuer: valid,
		},
		{
			name: "wrong signature",
			claims: Claims{RegisteredClaims: jwt.RegisteredClaims{
				Issuer: Issuer, Audience: []string{Audience},
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
			}},
			issuer: other,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			token, err := test.issuer.GenerateToken(test.claims)
			if err != nil {
				t.Fatalf("generate token: %v", err)
			}
			if _, err := valid.ValidateToken(token); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}
