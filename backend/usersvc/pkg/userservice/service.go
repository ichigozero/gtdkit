package userservice

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/usersvc"
)

type Service interface {
	UserID(ctx context.Context, username, password string) (uint64, error)
}

func New(u usersvc.UserRepository, logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService(u)
		svc = LoggingMiddleware(logger)(svc)
	}
	return svc
}

func NewBasicService(u usersvc.UserRepository) Service {
	return basicService{users: u}
}

type basicService struct {
	users usersvc.UserRepository
}

func (s basicService) UserID(_ context.Context, username, password string) (uint64, error) {
	if username == "" || password == "" {
		return 0, usersvc.ErrInvalidArgument
	}

	uid, err := s.users.UserID(username, password)
	if err != nil {
		return uid, err
	}

	return uid, nil
}
