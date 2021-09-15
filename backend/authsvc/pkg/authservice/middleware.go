package authservice

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
)

type Middleware func(Service) Service

func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) Login(ctx context.Context, username, password string) (tokens map[string]string, err error) {
	defer func() {
		mw.logger.Log("method", "Login", "err", err)
	}()
	return mw.next.Login(ctx, username, password)
}

func ProxingMiddleware(ctx context.Context, userIDEndpoint endpoint.Endpoint) Middleware {
	return func(next Service) Service {
		return proxingMiddleware{next, userIDEndpoint}
	}
}

type proxingMiddleware struct {
	next   Service
	userID endpoint.Endpoint
}

func (mw proxingMiddleware) Login(ctx context.Context, username, password string) (map[string]string, error) {
	response, err := mw.userID(ctx, userendpoint.UserIDRequest{Name: username, Password: password})
	if err != nil {
		return nil, err
	}

	resp := response.(userendpoint.UserIDResponse)
	if resp.Err != nil {
		return nil, resp.Err
	}

	ctx = context.WithValue(ctx, UserIDContextKey, resp.ID)

	return mw.next.Login(ctx, username, password)
}

const UserIDContextKey = "UserID"
