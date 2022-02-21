package userservice

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
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

func (mw loggingMiddleware) UserID(ctx context.Context, username, password string) (id uint64, err error) {
	defer func() {
		mw.logger.Log("method", "UserID", "username", username, "id", id, "err", err)
	}()
	return mw.next.UserID(ctx, username, password)
}

func (mw loggingMiddleware) IsExists(ctx context.Context, id uint64) (v bool, err error) {
	defer func() {
		mw.logger.Log("method", "IsExists", "id", id, "v", v, "err", err)
	}()
	return mw.next.IsExists(ctx, id)
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

func (mw instrumentingMiddleware) UserID(ctx context.Context, username, password string) (id uint64, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "user_id").Add(1)
		mw.requestLatency.With("method", "user_id").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.UserID(ctx, username, password)
}

func (mw instrumentingMiddleware) IsExists(ctx context.Context, id uint64) (v bool, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "is_exists").Add(1)
		mw.requestLatency.With("method", "is_exists").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.IsExists(ctx, id)
}
