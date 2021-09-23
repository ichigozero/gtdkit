package authservice

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"
)

type Service interface {
	Login(ctx context.Context, username, password string) (map[string]string, error)
}

func New(t Tokenizer, logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService(t)
		svc = LoggingMiddleware(logger)(svc)

	}
	return svc
}

type basicService struct {
	tokenizer Tokenizer
}

func NewBasicService(t Tokenizer) Service {
	return &basicService{tokenizer: t}
}

func (s *basicService) Login(ctx context.Context, _, _ string) (map[string]string, error) {
	userID, ok := ctx.Value(UserIDContextKey).(uint64)
	if !ok {
		return nil, ErrUserIDContextMissing
	}

	at, rt, err := s.tokenizer.Generate(userID)
	if err != nil {
		return nil, err
	}

	tokens := map[string]string{
		"access_token":  at.Hash,
		"refresh_token": rt.Hash,
	}

	return tokens, nil
}

var ErrUserIDContextMissing = errors.New("user ID was not passed through the context")
