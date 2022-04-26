package userservice

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/usersvc"
	"golang.org/x/crypto/bcrypt"
)

type Service interface {
	UserID(ctx context.Context, username, password string) (uint64, error)
	IsExists(ctx context.Context, id uint64) (bool, error)
}

func New(u usersvc.UserRepository, logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService(u)
		svc = LoggingMiddleware(logger)(svc)
	}
	return svc
}

type basicService struct {
	users usersvc.UserRepository
}

func NewBasicService(u usersvc.UserRepository) Service {
	return basicService{users: u}
}

func (s basicService) UserID(_ context.Context, username, password string) (uint64, error) {
	if username == "" || password == "" {
		return 0, usersvc.ErrInvalidArgument
	}

	u := s.users.GetUser(username)
	if u.ID == 0 {
		return 0, usersvc.ErrUserNotFound
	}

	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	if err != nil {
		return 0, usersvc.ErrUserNotFound
	}

	return u.ID, nil
}

func (s basicService) IsExists(_ context.Context, id uint64) (bool, error) {
	if id == 0 {
		return false, usersvc.ErrInvalidArgument
	}

	return s.users.IsExists(id)
}
