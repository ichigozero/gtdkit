package client

import (
	"io"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/go-kit/kit/sd/lb"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskendpoint"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskservice"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/tasktransport"
	"google.golang.org/grpc"
)

func New(apiclient consulsd.Client, logger log.Logger, retryMax int, retryTimeout time.Duration) (taskendpoint.Set, error) {
	var (
		tags        = []string{}
		passingOnly = true
		endpoints   = taskendpoint.Set{}
		instancer   = consulsd.NewInstancer(apiclient, logger, "tasksvc", tags, passingOnly)
	)
	{
		factory := factoryFor(taskendpoint.MakeCreateTaskEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.CreateTaskEndpoint = retry
	}
	{
		factory := factoryFor(taskendpoint.MakeTasksEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.TasksEndpoint = retry
	}
	{
		factory := factoryFor(taskendpoint.MakeTaskEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.TaskEndpoint = retry
	}
	{
		factory := factoryFor(taskendpoint.MakeUpdateTaskEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.UpdateTaskEndpoint = retry
	}
	{
		factory := factoryFor(taskendpoint.MakeDeleteTaskEndpoint, logger)
		endpointer := sd.NewEndpointer(instancer, factory, logger)
		balancer := lb.NewRoundRobin(endpointer)
		retry := lb.Retry(retryMax, retryTimeout, balancer)
		endpoints.DeleteTaskEndpoint = retry
	}
	return endpoints, nil
}

func factoryFor(makeEndpoint func(taskservice.Service) endpoint.Endpoint, logger log.Logger) sd.Factory {
	return func(instance string) (endpoint.Endpoint, io.Closer, error) {
		conn, err := grpc.Dial(instance, grpc.WithInsecure())
		if err != nil {
			return nil, nil, err
		}
		service := tasktransport.NewGRPCClient(conn, logger)
		endpoint := makeEndpoint(service)

		return endpoint, conn, nil
	}
}
