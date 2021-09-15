package client

import (
	"io"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/usertransport"
	"google.golang.org/grpc"
)

func New(apiclient consulsd.Client, logger log.Logger, retryMax int, retryTimeout time.Duration) (userendpoint.Set, error) {
	var (
		tags        = []string{}
		passingOnly = true
		endpoints   = userendpoint.Set{}
		instancer   = consulsd.NewInstancer(apiclient, logger, "usersvc", tags, passingOnly)
	)
	{
		factory := factoryFor(userendpoint.MakeUserIDEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.UserIDEndpoint = retry
	}

	return endpoints, nil
}

func factoryFor(makeEndpoint func(userservice.Service) endpoint.Endpoint, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := usertransport.NewGRPCClient(conn, logger)
		endpoint := makeEndpoint(service)

		return endpoint, conn, nil
	}
}
