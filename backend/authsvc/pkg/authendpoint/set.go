package authendpoint

import (
	"context"
	"fmt"
	"strconv"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/authsvc"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authservice"
)

type Set struct {
	LoginEndpoint   endpoint.Endpoint
	LogoutEndpoint  endpoint.Endpoint
	RefreshEndpoint endpoint.Endpoint
}

func New(svc authservice.Service, logger log.Logger) Set {
	var loginEndpoint endpoint.Endpoint
	{
		loginEndpoint = MakeLoginEndpoint(svc)
		loginEndpoint = LoggingMiddleware(log.With(logger, "method", "Login"))(loginEndpoint)
	}

	var logoutEndpoint endpoint.Endpoint
	{
		logoutEndpoint = MakeLogoutEndpoint(svc)
		logoutEndpoint = LoggingMiddleware(log.With(logger, "method", "Logout"))(logoutEndpoint)
	}

	var refreshEndpoint endpoint.Endpoint
	{
		refreshEndpoint = MakeRefreshEndpoint(svc)
		refreshEndpoint = LoggingMiddleware(log.With(logger, "method", "Refresh"))(refreshEndpoint)
	}

	return Set{
		LoginEndpoint:   loginEndpoint,
		LogoutEndpoint:  logoutEndpoint,
		RefreshEndpoint: refreshEndpoint,
	}
}

func (s Set) Login(ctx context.Context, username, password string) (map[string]string, error) {
	response, err := s.LoginEndpoint(ctx, LoginRequest{Username: username, Password: password})
	if err != nil {
		return nil, err
	}

	resp := response.(LoginResponse)
	return resp.Tokens, resp.Err
}

func (s Set) Logout(ctx context.Context, accessUUID string) (bool, error) {
	response, err := s.LogoutEndpoint(ctx, LogoutRequest{})
	if err != nil {
		return false, err
	}

	resp := response.(LogoutResponse)
	return resp.Success, resp.Err
}

func (s Set) Refresh(ctx context.Context, accessUUID, refreshUUID string, userID uint64) (map[string]string, error) {
	response, err := s.RefreshEndpoint(ctx, RefreshRequest{})
	if err != nil {
		return nil, err
	}

	resp := response.(RefreshResponse)
	return resp.Tokens, resp.Err
}

func MakeLoginEndpoint(s authservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(LoginRequest)
		t, err := s.Login(ctx, req.Username, req.Password)

		return LoginResponse{Tokens: t, Err: err}, nil
	}
}

func MakeLogoutEndpoint(s authservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		uuid, ok := ctx.Value(authsvc.JWTUUIDContextKey).(string)
		if !ok {
			return LogoutResponse{Err: authsvc.ErrUUIDMissing}, nil
		}

		_ = request.(LogoutRequest)
		s, err := s.Logout(ctx, uuid)

		return LogoutResponse{Success: s, Err: err}, nil
	}
}

func MakeRefreshEndpoint(s authservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		claims, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(stdjwt.MapClaims)
		if !ok {
			return nil, authsvc.ErrClaimsMissing
		}

		accessUUID, ok := claims["access_uuid"].(string)
		if !ok {
			return nil, authsvc.ErrClaimsInvalid
		}

		refreshUUID, ok := claims["refresh_uuid"].(string)
		if !ok {
			return nil, authsvc.ErrClaimsInvalid
		}

		userID, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
		if err != nil {
			return nil, authsvc.ErrClaimsInvalid
		}

		_ = request.(RefreshRequest)
		t, err := s.Refresh(ctx, accessUUID, refreshUUID, userID)

		return RefreshResponse{Tokens: t, Err: err}, nil
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Tokens map[string]string `json:"tokens"`
	Err    error             `json:"-"`
}

func (r LoginResponse) Failed() error { return r.Err }

type LogoutRequest struct{}

type LogoutResponse struct {
	Success bool  `json:"success"`
	Err     error `json:"-"`
}

func (r LogoutResponse) Failed() error { return r.Err }

type RefreshRequest struct{}

type RefreshResponse struct {
	Tokens map[string]string `json:"tokens"`
	Err    error             `json:"-"`
}

func (r RefreshResponse) Failed() error { return r.Err }
