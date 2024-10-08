package authservice

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/authsvc"
	"github.com/ichigozero/gtdkit/backend/authsvc/inmem"
	stduuid "github.com/twinj/uuid"
)

type Service interface {
	Login(ctx context.Context, username, password string) (map[string]string, error)
	Logout(ctx context.Context, accessUUID string) (bool, error)
	Refresh(ctx context.Context, accessUUID, refreshUUID string, userID uint64) (map[string]string, error)
	Validate(ctx context.Context, accessUUID string) (bool, error)
}

func New(t Tokenizer, c inmem.Client, logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService(t, c)
		svc = LoggingMiddleware(logger)(svc)

	}
	return svc
}

type basicService struct {
	tokenizer Tokenizer
	client    inmem.Client
}

func NewBasicService(t Tokenizer, c inmem.Client) Service {
	return &basicService{tokenizer: t, client: c}
}

func (s *basicService) Login(ctx context.Context, _, _ string) (map[string]string, error) {
	userID, ok := ctx.Value(authsvc.UserIDContextKey).(uint64)
	if !ok {
		return nil, authsvc.ErrUserIDContextMissing
	}

	at, rt, err := s.tokenizer.Generate(userID)
	if err != nil {
		return nil, err
	}

	s.storeTokens(at, rt)

	return s.compileTokens(at, rt), nil
}

func (s *basicService) Logout(ctx context.Context, accessUUID string) (bool, error) {
	if _, err := s.validate(ctx, accessUUID); err != nil {
		return false, err
	}

	ruuid := stduuid.NewV5(stduuid.NameSpaceURL, accessUUID).String()

	var err error
	{
		err = s.client.Delete(accessUUID)
		if err != nil {
			return false, err
		}
		err = s.client.Delete(ruuid)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

func (s *basicService) Refresh(_ context.Context, accessUUID, refreshUUID string, userID uint64) (map[string]string, error) {
	if accessUUID == "" || refreshUUID == "" || userID == 0 {
		return nil, authsvc.ErrInvalidArgument
	}

	if err := s.client.Get(refreshUUID); err != nil {
		return nil, err
	}

	var err error
	{
		err = s.client.Delete(accessUUID)
		if err != nil {
			return nil, err
		}

		err = s.client.Delete(refreshUUID)
		if err != nil {
			return nil, err
		}
	}

	at, rt, err := s.tokenizer.Generate(userID)
	if err != nil {
		return nil, err
	}

	s.storeTokens(at, rt)

	return s.compileTokens(at, rt), nil
}

func (s *basicService) Validate(ctx context.Context, accessUUID string) (bool, error) {
	return s.validate(ctx, accessUUID)
}

func (s *basicService) storeTokens(at *AccessToken, rt *RefreshToken) {
	s.client.Put(at.UUID, []byte(at.Hash))
	s.client.Put(rt.RefreshUUID, []byte(rt.Hash))
}

func (s *basicService) compileTokens(at *AccessToken, rt *RefreshToken) map[string]string {
	return map[string]string{
		"access":  at.Hash,
		"refresh": rt.Hash,
	}
}

func (s *basicService) validate(_ context.Context, accessUUID string) (bool, error) {
	if accessUUID == "" {
		return false, authsvc.ErrInvalidArgument
	}

	if err := s.client.Get(accessUUID); err != nil {
		return false, err
	}
	return true, nil
}
