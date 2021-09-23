package authtransport

import (
	"context"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/ichigozero/gtdkit/backend/authsvc"
	"github.com/ichigozero/gtdkit/backend/authsvc/inmem"
)

func NewAuthenticater(c inmem.Client) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			claims, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(stdjwt.MapClaims)
			if !ok {
				return nil, authsvc.ErrClaimsMissing
			}

			uuid, ok := claims["uuid"].(string)
			if !ok {
				return nil, authsvc.ErrUUIDMissing
			}

			err = c.Get(uuid)
			if err != nil {
				return nil, err
			}

			ctx = context.WithValue(ctx, authsvc.JWTUUIDContextKey, uuid)

			return next(ctx, request)
		}
	}
}
