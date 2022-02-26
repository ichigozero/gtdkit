package tasktransport

import (
	"context"
	"errors"
	"os"
	"time"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/circuitbreaker"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/ratelimit"
	"github.com/go-kit/kit/transport"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pb"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskendpoint"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskservice"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
)

type grpcServer struct {
	createTask grpctransport.Handler
	tasks      grpctransport.Handler
	task       grpctransport.Handler
	updateTask grpctransport.Handler
	deleteTask grpctransport.Handler
	pb.UnimplementedTaskSVCServer
}

func NewGRPCServer(endpoints taskendpoint.Set, logger log.Logger) pb.TaskSVCServer {
	options := []grpctransport.ServerOption{
		grpctransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	kf := func(token *stdjwt.Token) (interface{}, error) {
		return []byte(os.Getenv("ACCESS_SECRET")), nil
	}

	var createTaskEndpoint endpoint.Endpoint
	{
		createTaskEndpoint = endpoints.CreateTaskEndpoint
		createTaskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(createTaskEndpoint)
	}

	var tasksEndpoint endpoint.Endpoint
	{
		tasksEndpoint = endpoints.TasksEndpoint
		tasksEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(tasksEndpoint)
	}

	var taskEndpoint endpoint.Endpoint
	{
		taskEndpoint = endpoints.TaskEndpoint
		taskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(taskEndpoint)
	}

	var updateTaskEndpoint endpoint.Endpoint
	{
		updateTaskEndpoint = endpoints.UpdateTaskEndpoint
		updateTaskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(updateTaskEndpoint)
	}

	var deleteTaskEndpoint endpoint.Endpoint
	{
		deleteTaskEndpoint = endpoints.DeleteTaskEndpoint
		deleteTaskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(deleteTaskEndpoint)
	}

	return &grpcServer{
		createTask: grpctransport.NewServer(
			createTaskEndpoint,
			decodeGRPCCreateTaskRequest,
			encodeGRPCCreateTaskResponse,
			append(options, grpctransport.ServerBefore(kitjwt.GRPCToContext()))...,
		),
		tasks: grpctransport.NewServer(
			tasksEndpoint,
			decodeGRPCTasksRequest,
			encodeGRPCTasksResponse,
			append(options, grpctransport.ServerBefore(kitjwt.GRPCToContext()))...,
		),
		task: grpctransport.NewServer(
			taskEndpoint,
			decodeGRPCTaskRequest,
			encodeGRPCTaskResponse,
			append(options, grpctransport.ServerBefore(kitjwt.GRPCToContext()))...,
		),
		updateTask: grpctransport.NewServer(
			updateTaskEndpoint,
			decodeGRPCUpdateTaskRequest,
			encodeGRPCUpdateTaskResponse,
			append(options, grpctransport.ServerBefore(kitjwt.GRPCToContext()))...,
		),
		deleteTask: grpctransport.NewServer(
			deleteTaskEndpoint,
			decodeGRPCDeleteTaskRequest,
			encodeGRPCDeleteTaskResponse,
			append(options, grpctransport.ServerBefore(kitjwt.GRPCToContext()))...,
		),
	}
}

func (s *grpcServer) CreateTask(ctx context.Context, req *pb.CreateTaskRequest) (*pb.CreateTaskReply, error) {
	_, rep, err := s.createTask.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.CreateTaskReply), nil
}

func (s *grpcServer) Tasks(ctx context.Context, req *pb.TasksRequest) (*pb.TasksReply, error) {
	_, rep, err := s.tasks.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.TasksReply), nil
}

func (s *grpcServer) Task(ctx context.Context, req *pb.TaskRequest) (*pb.TaskReply, error) {
	_, rep, err := s.task.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.TaskReply), nil
}

func (s *grpcServer) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.UpdateTaskReply, error) {
	_, rep, err := s.updateTask.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.UpdateTaskReply), nil
}

func (s *grpcServer) DeleteTask(ctx context.Context, req *pb.DeleteTaskRequest) (*pb.DeleteTaskReply, error) {
	_, rep, err := s.deleteTask.ServeGRPC(ctx, req)
	if err != nil {
		return nil, err
	}
	return rep.(*pb.DeleteTaskReply), nil
}

func NewGRPCClient(conn *grpc.ClientConn, logger log.Logger) taskservice.Service {
	limiter := ratelimit.NewErroringLimiter(rate.NewLimiter(rate.Every(time.Second), 100))

	var options []grpctransport.ClientOption

	var createTaskEndpoint endpoint.Endpoint
	{
		createTaskEndpoint = grpctransport.NewClient(
			conn,
			"pb.TaskSVC",
			"CreateTask",
			encodeGRPCCreateTaskRequest,
			decodeGRPCCreateTaskResponse,
			pb.CreateTaskReply{},
			append(options, grpctransport.ClientBefore(kitjwt.ContextToGRPC()))...,
		).Endpoint()
		createTaskEndpoint = limiter(createTaskEndpoint)
		createTaskEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "CreateTask",
			Timeout: 30 * time.Second,
		}))(createTaskEndpoint)
	}

	var tasksEndpoint endpoint.Endpoint
	{
		tasksEndpoint = grpctransport.NewClient(
			conn,
			"pb.TaskSVC",
			"Tasks",
			encodeGRPCTasksRequest,
			decodeGRPCTasksResponse,
			pb.TasksReply{},
			append(options, grpctransport.ClientBefore(kitjwt.ContextToGRPC()))...,
		).Endpoint()
		tasksEndpoint = limiter(tasksEndpoint)
		tasksEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Tasks",
			Timeout: 30 * time.Second,
		}))(tasksEndpoint)
	}

	var taskEndpoint endpoint.Endpoint
	{
		taskEndpoint = grpctransport.NewClient(
			conn,
			"pb.TaskSVC",
			"Task",
			encodeGRPCTaskRequest,
			decodeGRPCTaskResponse,
			pb.TaskReply{},
			append(options, grpctransport.ClientBefore(kitjwt.ContextToGRPC()))...,
		).Endpoint()
		taskEndpoint = limiter(taskEndpoint)
		taskEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "Task",
			Timeout: 30 * time.Second,
		}))(taskEndpoint)
	}

	var updateTaskEndpoint endpoint.Endpoint
	{
		updateTaskEndpoint = grpctransport.NewClient(
			conn,
			"pb.TaskSVC",
			"UpdateTask",
			encodeGRPCUpdateTaskRequest,
			decodeGRPCUpdateTaskResponse,
			pb.UpdateTaskReply{},
			append(options, grpctransport.ClientBefore(kitjwt.ContextToGRPC()))...,
		).Endpoint()
		updateTaskEndpoint = limiter(updateTaskEndpoint)
		updateTaskEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "UpdateTask",
			Timeout: 30 * time.Second,
		}))(updateTaskEndpoint)
	}

	var deleteTaskEndpoint endpoint.Endpoint
	{
		deleteTaskEndpoint = grpctransport.NewClient(
			conn,
			"pb.TaskSVC",
			"DeleteTask",
			encodeGRPCDeleteTaskRequest,
			decodeGRPCDeleteTaskResponse,
			pb.DeleteTaskReply{},
			append(options, grpctransport.ClientBefore(kitjwt.ContextToGRPC()))...,
		).Endpoint()
		deleteTaskEndpoint = limiter(deleteTaskEndpoint)
		deleteTaskEndpoint = circuitbreaker.Gobreaker(gobreaker.NewCircuitBreaker(gobreaker.Settings{
			Name:    "DeleteTask",
			Timeout: 30 * time.Second,
		}))(deleteTaskEndpoint)
	}

	return taskendpoint.Set{
		CreateTaskEndpoint: createTaskEndpoint,
		TasksEndpoint:      tasksEndpoint,
		TaskEndpoint:       taskEndpoint,
		UpdateTaskEndpoint: updateTaskEndpoint,
		DeleteTaskEndpoint: deleteTaskEndpoint,
	}
}

func decodeGRPCCreateTaskRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.CreateTaskRequest)
	return taskendpoint.CreateTaskRequest{
		Title:       req.Title,
		Description: req.Description,
	}, nil
}

func encodeGRPCCreateTaskResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(taskendpoint.CreateTaskResponse)
	return &pb.CreateTaskReply{
		Task: &pb.Task{
			Id:          resp.Task.ID,
			Title:       resp.Task.Title,
			Description: resp.Task.Description,
			Done:        resp.Task.Done,
			UserId:      resp.Task.UserID,
		},
		Err: err2str(resp.Err),
	}, nil
}

func encodeGRPCCreateTaskRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(taskendpoint.CreateTaskRequest)
	return &pb.CreateTaskRequest{
		Title:       req.Title,
		Description: req.Description,
	}, nil
}

func decodeGRPCCreateTaskResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.CreateTaskReply)
	return taskendpoint.CreateTaskResponse{
		Task: tasksvc.Task{
			ID:          reply.Task.Id,
			Title:       reply.Task.Title,
			Description: reply.Task.Description,
			Done:        reply.Task.Done,
			UserID:      reply.Task.UserId,
		},
		Err: str2err(reply.Err),
	}, nil
}

func decodeGRPCTasksRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	return taskendpoint.TasksRequest{}, nil
}

func encodeGRPCTasksResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(taskendpoint.TasksResponse)
	var tasks []*pb.Task
	for _, t := range resp.Tasks {
		task := &pb.Task{
			Id:          t.ID,
			Title:       t.Title,
			Description: t.Description,
			Done:        t.Done,
			UserId:      t.UserID,
		}
		tasks = append(tasks, task)
	}

	return &pb.TasksReply{
		Tasks: tasks,
		Err:   err2str(resp.Err),
	}, nil
}

func encodeGRPCTasksRequest(_ context.Context, request interface{}) (interface{}, error) {
	return &pb.TasksRequest{}, nil
}

func decodeGRPCTasksResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.TasksReply)
	var tasks []tasksvc.Task
	for _, t := range reply.Tasks {
		task := tasksvc.Task{
			ID:          t.Id,
			Title:       t.Title,
			Description: t.Description,
			Done:        t.Done,
			UserID:      t.UserId,
		}
		tasks = append(tasks, task)
	}

	return taskendpoint.TasksResponse{
		Tasks: tasks,
		Err:   str2err(reply.Err),
	}, nil
}

func decodeGRPCTaskRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.TaskRequest)
	return taskendpoint.TaskRequest{
		TaskID: req.TaskId,
	}, nil
}

func encodeGRPCTaskResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(taskendpoint.TaskResponse)
	return &pb.TaskReply{
		Task: &pb.Task{
			Id:          resp.Task.ID,
			Title:       resp.Task.Title,
			Description: resp.Task.Description,
			Done:        resp.Task.Done,
			UserId:      resp.Task.UserID,
		},
		Err: err2str(resp.Err),
	}, nil
}

func encodeGRPCTaskRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(taskendpoint.TaskRequest)
	return &pb.TaskRequest{
		TaskId: req.TaskID,
	}, nil
}

func decodeGRPCTaskResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.TaskReply)
	return taskendpoint.TaskResponse{
		Task: tasksvc.Task{
			ID:          reply.Task.Id,
			Title:       reply.Task.Title,
			Description: reply.Task.Description,
			Done:        reply.Task.Done,
			UserID:      reply.Task.UserId,
		},
		Err: str2err(reply.Err),
	}, nil
}

func decodeGRPCUpdateTaskRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.UpdateTaskRequest)
	return taskendpoint.UpdateTaskRequest{
		TaskID:      req.Id,
		Title:       req.Title,
		Description: req.Description,
		Done:        req.Done,
	}, nil
}

func encodeGRPCUpdateTaskResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(taskendpoint.UpdateTaskResponse)
	return &pb.UpdateTaskReply{
		Task: &pb.Task{
			Id:          resp.Task.ID,
			Title:       resp.Task.Title,
			Description: resp.Task.Description,
			Done:        resp.Task.Done,
		},
		Err: err2str(resp.Err),
	}, nil
}

func encodeGRPCUpdateTaskRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(taskendpoint.UpdateTaskRequest)
	return &pb.UpdateTaskRequest{
		Id:          req.TaskID,
		Title:       req.Title,
		Description: req.Description,
		Done:        req.Done,
	}, nil
}

func decodeGRPCUpdateTaskResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.UpdateTaskReply)
	return taskendpoint.UpdateTaskResponse{
		Task: tasksvc.Task{
			ID:          reply.Task.Id,
			Title:       reply.Task.Title,
			Description: reply.Task.Description,
			Done:        reply.Task.Done,
		},
		Err: str2err(reply.Err),
	}, nil
}

func decodeGRPCDeleteTaskRequest(_ context.Context, grpcReq interface{}) (interface{}, error) {
	req := grpcReq.(*pb.DeleteTaskRequest)
	return taskendpoint.DeleteTaskRequest{
		TaskID: req.TaskId,
	}, nil
}

func encodeGRPCDeleteTaskResponse(_ context.Context, response interface{}) (interface{}, error) {
	resp := response.(taskendpoint.DeleteTaskResponse)
	return &pb.DeleteTaskReply{
		Result: resp.Result,
		Err:    err2str(resp.Err),
	}, nil
}

func encodeGRPCDeleteTaskRequest(_ context.Context, request interface{}) (interface{}, error) {
	req := request.(taskendpoint.DeleteTaskRequest)
	return &pb.DeleteTaskRequest{
		TaskId: req.TaskID,
	}, nil
}

func decodeGRPCDeleteTaskResponse(_ context.Context, grpcReply interface{}) (interface{}, error) {
	reply := grpcReply.(*pb.DeleteTaskReply)
	return taskendpoint.DeleteTaskResponse{
		Result: reply.Result,
		Err:    str2err(reply.Err),
	}, nil
}

func str2err(s string) error {
	if s == "" {
		return nil
	}

	switch s {
	case tasksvc.ErrInvalidArgument.Error():
		return tasksvc.ErrInvalidArgument
	}

	return errors.New(s)
}

func err2str(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
