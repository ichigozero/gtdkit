package authendpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authservice"
)

type Set struct {
	LoginEndpoint endpoint.Endpoint
}

func New(svc authservice.Service, logger log.Logger) Set {
	var loginEndpoint endpoint.Endpoint
	{
		loginEndpoint = MakeLoginEndpoint(svc)
		loginEndpoint = LoggingMiddleware(log.With(logger, "method", "Login"))(loginEndpoint)
	}

	return Set{
		LoginEndpoint: loginEndpoint,
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

func MakeLoginEndpoint(s authservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(LoginRequest)
		t, err := s.Login(ctx, req.Username, req.Password)
		return LoginResponse{Tokens: t, Err: err}, nil
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Tokens map[string]string `json:"tokens"`
	Err    error             `json:"-"` // should be intercepted by Failed/errorEncoder
}

func (r LoginResponse) Failed() error { return r.Err }
