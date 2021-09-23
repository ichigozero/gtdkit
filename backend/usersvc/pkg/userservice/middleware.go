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
		mw.logger.Log("method", "User", "username", username, "err", err)
	}()
	return mw.next.UserID(ctx, username, password)
}
