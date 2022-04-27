package authsvc

import (
	"errors"
	"os"
)

var (
	AppEnv         = getEnv("APP_ENV", "")
	AccessSecret   = getEnv("ACCESS_SECRET", "access-secret")
	RefreshSecret  = getEnv("REFRESH_SECRET", "refresh-secret")
	CookieHashKey  = getEnv("COOKIE_HASH_KEY", "very-secret")
	CookieBlockKey = getEnv("COOKIE_BLOCK_KEY", "a-lots-of-secret")
)

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

type contextKey string

const (
	UserIDContextKey  contextKey = "UserID"
	JWTUUIDContextKey contextKey = "JWTUUID"
)

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrUserIDContextMissing = errors.New("user ID was not passed through the context")
	ErrClaimsMissing        = errors.New("JWT claims was not passed through the context")
	ErrClaimsInvalid        = errors.New("JWT claims was invalid")
)
