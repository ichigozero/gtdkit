package authsvc

import "errors"

type contextKey string

const UserIDContextKey contextKey = "UserID"
const JWTUUIDContextKey contextKey = "JWTUUID"

var (
	ErrInvalidArgument      = errors.New("invalid argument")
	ErrUserIDContextMissing = errors.New("user ID was not passed through the context")
	ErrClaimsMissing        = errors.New("JWT claims was not passed through the context")
	ErrClaimsInvalid        = errors.New("JWT claims was invalid")
	ErrUUIDMissing          = errors.New("JWT UUID is missing from JWT claims")
)
