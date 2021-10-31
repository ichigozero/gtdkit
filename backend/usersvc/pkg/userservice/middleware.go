package userservice

import (
	"context"

	"github.com/go-kit/kit/log"
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
