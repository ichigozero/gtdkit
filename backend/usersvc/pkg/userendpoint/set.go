package userendpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
)

type Set struct {
	UserIDEndpoint endpoint.Endpoint
}

func New(svc userservice.Service, logger log.Logger) Set {
	var userIDEndpoint endpoint.Endpoint
	{
		userIDEndpoint = MakeUserIDEndpoint(svc)
		userIDEndpoint = LoggingMiddleware(log.With(logger, "method", "User"))(userIDEndpoint)
	}
	return Set{
		UserIDEndpoint: userIDEndpoint,
	}
}

func MakeUserIDEndpoint(s userservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UserIDRequest)
		id, err := s.UserID(ctx, req.Name, req.Password)
		return UserIDResponse{ID: id, Err: err}, nil
	}
}

type UserIDRequest struct {
	Name, Password string
}

type UserIDResponse struct {
	ID  int   `json:"id"`
	Err error `json:"err"`
}
