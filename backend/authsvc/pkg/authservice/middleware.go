package authservice

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
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

func (mw loggingMiddleware) Validate(ctx context.Context, accessUUID string) (v bool, err error) {
	defer func() {
		mw.logger.Log("method", "Validate", "access_uuid", accessUUID, "v", v, "err", err)
	}()
	return mw.next.Validate(ctx, accessUUID)
}

func ProxingMiddleware(ctx context.Context, userIDEndpoint, isUserExists endpoint.Endpoint) Middleware {
	return func(next Service) Service {
		return proxingMiddleware{next, userIDEndpoint, isUserExists}
	}
}

func InstrumentingMiddleware(counter metrics.Counter, latency metrics.Histogram, s Service) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{counter, latency, next}
	}
}

type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           Service
}

func (mw instrumentingMiddleware) Login(ctx context.Context, username, password string) (tokens map[string]string, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "login").Add(1)
		mw.requestLatency.With("method", "login").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Login(ctx, username, password)
}

func (mw instrumentingMiddleware) Logout(ctx context.Context, accessUUID string) (success bool, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "logout").Add(1)
		mw.requestLatency.With("method", "logout").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Logout(ctx, accessUUID)
}

func (mw instrumentingMiddleware) Refresh(ctx context.Context, accessUUID, refreshUUID string, userID uint64) (tokens map[string]string, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "refresh").Add(1)
		mw.requestLatency.With("method", "refresh").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Refresh(ctx, accessUUID, refreshUUID, userID)
}

func (mw instrumentingMiddleware) Validate(ctx context.Context, accessUUID string) (v bool, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "validate").Add(1)
		mw.requestLatency.With("method", "validate").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Validate(ctx, accessUUID)
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

func (mw proxingMiddleware) Validate(ctx context.Context, accessUUID string) (bool, error) {
	return mw.next.Validate(ctx, accessUUID)
}
