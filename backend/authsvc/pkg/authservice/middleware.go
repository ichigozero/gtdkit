package authservice

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/authsvc"
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

func (mw loggingMiddleware) Logout(ctx context.Context, accessUUID string) (success bool, err error) {
	defer func() {
		mw.logger.Log("method", "Logout", "success", success, "err", err)
	}()
	return mw.next.Logout(ctx, accessUUID)
}

func (mw loggingMiddleware) Refresh(ctx context.Context, accessUUID, refreshUUID string, userID uint64) (tokens map[string]string, err error) {
	defer func() {
		mw.logger.Log("method", "Refresh", "err", err)
	}()
	return mw.next.Refresh(ctx, accessUUID, refreshUUID, userID)
}

func ProxingMiddleware(ctx context.Context, userIDEndpoint, isUserExists endpoint.Endpoint) Middleware {
	return func(next Service) Service {
		return proxingMiddleware{next, userIDEndpoint, isUserExists}
	}
}

type proxingMiddleware struct {
	next         Service
	userID       endpoint.Endpoint
	isUserExists endpoint.Endpoint
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

	ctx = context.WithValue(ctx, authsvc.UserIDContextKey, resp.ID)

	return mw.next.Login(ctx, username, password)
}

func (mw proxingMiddleware) Logout(ctx context.Context, accessUUID string) (bool, error) {
	return mw.next.Logout(ctx, accessUUID)
}

func (mw proxingMiddleware) Refresh(ctx context.Context, accessUUID, refreshUUID string, userID uint64) (map[string]string, error) {
	response, err := mw.isUserExists(ctx, userendpoint.IsExistsRequest{ID: userID})
	if err != nil {
		return nil, err
	}

	resp := response.(userendpoint.IsExistsResponse)
	if resp.Err != nil {
		return nil, resp.Err
	}

	return mw.next.Refresh(ctx, accessUUID, refreshUUID, userID)
}
