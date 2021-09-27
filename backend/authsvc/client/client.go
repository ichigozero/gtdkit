package client

import (
	"io"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authendpoint"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authservice"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authtransport"
)

func New(apiclient consulsd.Client, logger log.Logger, retryMax int, retryTimeout time.Duration) (authendpoint.Set, error) {
	var (
		tags        = []string{}
		passingOnly = true
		endpoints   = authendpoint.Set{}
		instancer   = consulsd.NewInstancer(apiclient, logger, "authsvc", tags, passingOnly)
	)
	{
		factory := factoryFor(authendpoint.MakeLoginEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.LoginEndpoint = retry
	}
	{
		factory := factoryFor(authendpoint.MakeLogoutEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.LogoutEndpoint = retry
	}
	{
		factory := factoryFor(authendpoint.MakeRefreshEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.RefreshEndpoint = retry
	}

	return endpoints, nil
}

func factoryFor(makeEndpoint func(authservice.Service) endpoint.Endpoint, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		service, err := authtransport.NewHTTPClient(instance, logger)
		if err != nil {
			return nil, nil, err
		}
		return makeEndpoint(service), nil, nil
	}
}
