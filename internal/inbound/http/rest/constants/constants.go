package constants

import "github.com/rickferrdev/sublyra-api/internal/platform/jwttoken"

type (
	SubscriptionAuthKey string
	SubscriptionAuth    struct {
		Claims *jwttoken.Claims
		Token  string
	}
)

const SUBSCRIPTION_AUTH_KEY = "auth_token"
