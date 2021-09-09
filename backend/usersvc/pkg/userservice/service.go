package userservice

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"
)

type Service interface {
	User(ctx context.Context, username, password string) (int, error)
}

func New(logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService()
		svc = LoggingMiddleware(logger)(svc)
	}
	return svc
}

func NewBasicService() Service {
	return basicService{}
}

type basicService struct{}

func (s basicService) User(_ context.Context, username, password string) (int, error) {
	if username == "" || password == "" {
		return 0, ErrInvalidArgument
	}

	if username != "admin" || password != "password" {
		return 0, ErrUserNotFound
	}

	return 1, nil
}

var (
	ErrInvalidArgument = errors.New("invalid argument")
	ErrUserNotFound    = errors.New("user not found")
)
