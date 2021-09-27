package userendpoint

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
)

type Set struct {
	UserIDEndpoint   endpoint.Endpoint
	IsExistsEndpoint endpoint.Endpoint
}

func New(svc userservice.Service, logger log.Logger) Set {
	var userIDEndpoint endpoint.Endpoint
	{
		userIDEndpoint = MakeUserIDEndpoint(svc)
		userIDEndpoint = LoggingMiddleware(log.With(logger, "method", "UserID"))(userIDEndpoint)
	}

	var isExistsEndpoint endpoint.Endpoint
	{
		isExistsEndpoint = MakeIsExistsEndpoint(svc)
		isExistsEndpoint = LoggingMiddleware(log.With(logger, "method", "IsExists"))(isExistsEndpoint)
	}

	return Set{
		UserIDEndpoint:   userIDEndpoint,
		IsExistsEndpoint: isExistsEndpoint,
	}
}

func (s Set) UserID(ctx context.Context, name, password string) (uint64, error) {
	resp, err := s.UserIDEndpoint(ctx, UserIDRequest{Name: name, Password: password})
	if err != nil {
		return 0, err
	}
	response := resp.(UserIDResponse)
	return response.ID, response.Err
}

func (s Set) IsExists(ctx context.Context, id uint64) (bool, error) {
	resp, err := s.IsExistsEndpoint(ctx, IsExistsRequest{ID: id})
	if err != nil {
		return false, err
	}
	response := resp.(IsExistsResponse)
	return response.V, response.Err
}

func MakeUserIDEndpoint(s userservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(UserIDRequest)
		id, err := s.UserID(ctx, req.Name, req.Password)
		return UserIDResponse{ID: id, Err: err}, nil
	}
}

func MakeIsExistsEndpoint(s userservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		req := request.(IsExistsRequest)
		v, err := s.IsExists(ctx, req.ID)
		return IsExistsResponse{V: v, Err: err}, nil
	}
}

type UserIDRequest struct {
	Name, Password string
}

type UserIDResponse struct {
	ID  uint64 `json:"id"`
	Err error  `json:"-"`
}

func (r UserIDResponse) Failed() error { return r.Err }

type IsExistsRequest struct {
	ID uint64
}

type IsExistsResponse struct {
	V   bool  `json:"v"`
	Err error `json:"-"`
}

func (r IsExistsResponse) Failed() error { return r.Err }
