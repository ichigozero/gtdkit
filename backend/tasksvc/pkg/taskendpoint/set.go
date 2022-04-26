package taskendpoint

import (
	"context"
	"fmt"
	"strconv"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskservice"
)

type Set struct {
	CreateTaskEndpoint endpoint.Endpoint
	TasksEndpoint      endpoint.Endpoint
	TaskEndpoint       endpoint.Endpoint
	UpdateTaskEndpoint endpoint.Endpoint
	DeleteTaskEndpoint endpoint.Endpoint
}

func New(svc taskservice.Service, logger log.Logger) Set {
	var createTaskEndpoint endpoint.Endpoint
	{
		createTaskEndpoint = MakeCreateTaskEndpoint(svc)
		createTaskEndpoint = LoggingMiddleware(log.With(logger, "method", "CreateTask"))(createTaskEndpoint)
	}
	var tasksEndpoint endpoint.Endpoint
	{
		tasksEndpoint = MakeTasksEndpoint(svc)
		tasksEndpoint = LoggingMiddleware(log.With(logger, "method", "Tasks"))(tasksEndpoint)
	}
	var taskEndpoint endpoint.Endpoint
	{
		taskEndpoint = MakeTaskEndpoint(svc)
		taskEndpoint = LoggingMiddleware(log.With(logger, "method", "Task"))(taskEndpoint)
	}

	var updateTaskEndpoint endpoint.Endpoint
	{
		updateTaskEndpoint = MakeUpdateTaskEndpoint(svc)
		updateTaskEndpoint = LoggingMiddleware(log.With(logger, "method", "UpdateTask"))(updateTaskEndpoint)
	}

	var deleteTaskEndpoint endpoint.Endpoint
	{
		deleteTaskEndpoint = MakeDeleteTaskEndpoint(svc)
		deleteTaskEndpoint = LoggingMiddleware(log.With(logger, "method", "DeleteTask"))(deleteTaskEndpoint)
	}

	return Set{
		CreateTaskEndpoint: createTaskEndpoint,
		TasksEndpoint:      tasksEndpoint,
		TaskEndpoint:       taskEndpoint,
		UpdateTaskEndpoint: updateTaskEndpoint,
		DeleteTaskEndpoint: deleteTaskEndpoint,
	}
}

func (s Set) CreateTask(ctx context.Context, a tasksvc.Auth, title, description string) (tasksvc.Task, error) {
	resp, err := s.CreateTaskEndpoint(ctx, CreateTaskRequest{Title: title, Description: description})
	if err != nil {
		return tasksvc.Task{}, err
	}
	response := resp.(CreateTaskResponse)
	return response.Task, response.Err
}

func (s Set) Tasks(ctx context.Context, a tasksvc.Auth) ([]tasksvc.Task, error) {
	resp, err := s.TasksEndpoint(ctx, TasksRequest{})
	if err != nil {
		return nil, err
	}
	response := resp.(TasksResponse)
	return response.Tasks, response.Err
}

func (s Set) Task(ctx context.Context, a tasksvc.Auth, taskID uint64) (tasksvc.Task, error) {
	resp, err := s.TaskEndpoint(ctx, TaskRequest{TaskID: taskID})
	if err != nil {
		return tasksvc.Task{}, err
	}
	response := resp.(TaskResponse)
	return response.Task, response.Err
}

func (s Set) UpdateTask(ctx context.Context, a tasksvc.Auth, task tasksvc.Task) (tasksvc.Task, error) {
	resp, err := s.UpdateTaskEndpoint(
		ctx,
		UpdateTaskRequest{
			TaskID:      task.ID,
			Title:       task.Title,
			Description: task.Description,
			Done:        task.Done,
		},
	)
	if err != nil {
		return tasksvc.Task{}, err
	}
	response := resp.(UpdateTaskResponse)
	return response.Task, response.Err
}

func (s Set) DeleteTask(ctx context.Context, a tasksvc.Auth, taskID uint64) (bool, error) {
	resp, err := s.DeleteTaskEndpoint(ctx, DeleteTaskRequest{TaskID: taskID})
	if err != nil {
		return false, err
	}
	response := resp.(DeleteTaskResponse)
	return response.Result, response.Err
}

func MakeCreateTaskEndpoint(s taskservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		auth, err := claims(ctx)
		if err != nil {
			return CreateTaskResponse{Err: err}, nil
		}

		req := request.(CreateTaskRequest)
		t, err := s.CreateTask(ctx, auth, req.Title, req.Description)
		return CreateTaskResponse{Task: t, Err: err}, nil
	}
}

func MakeTasksEndpoint(s taskservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		auth, err := claims(ctx)
		if err != nil {
			return TasksResponse{Err: err}, nil
		}

		_ = request.(TasksRequest)
		t, err := s.Tasks(ctx, auth)
		return TasksResponse{Tasks: t, Err: err}, nil
	}
}

func MakeTaskEndpoint(s taskservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		auth, err := claims(ctx)
		if err != nil {
			return TaskResponse{Err: err}, nil
		}

		req := request.(TaskRequest)
		t, err := s.Task(ctx, auth, req.TaskID)
		return TaskResponse{Task: t, Err: err}, nil
	}
}

func MakeUpdateTaskEndpoint(s taskservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		auth, err := claims(ctx)
		if err != nil {
			return UpdateTaskResponse{Err: err}, nil
		}

		req := request.(UpdateTaskRequest)
		t, err := s.UpdateTask(
			ctx,
			auth,
			tasksvc.Task{
				ID:          req.TaskID,
				Title:       req.Title,
				Description: req.Description,
				Done:        req.Done,
				UserID:      auth.UserID,
			},
		)
		return UpdateTaskResponse{Task: t, Err: err}, nil
	}
}

func MakeDeleteTaskEndpoint(s taskservice.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		auth, err := claims(ctx)
		if err != nil {
			return DeleteTaskResponse{Err: err}, nil
		}

		req := request.(DeleteTaskRequest)
		r, err := s.DeleteTask(ctx, auth, req.TaskID)
		return DeleteTaskResponse{Result: r, Err: err}, nil
	}
}

func claims(ctx context.Context) (tasksvc.Auth, error) {
	claims, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(stdjwt.MapClaims)
	if !ok {
		return tasksvc.Auth{}, tasksvc.ErrClaimsMissing
	}

	uuid, ok := claims["uuid"].(string)
	if !ok {
		return tasksvc.Auth{}, tasksvc.ErrClaimsMissing
	}

	userID, err := strconv.ParseUint(fmt.Sprintf("%.f", claims["user_id"]), 10, 64)
	if err != nil {
		return tasksvc.Auth{}, tasksvc.ErrClaimsMissing
	}

	return tasksvc.Auth{AccessUUID: uuid, UserID: userID}, nil
}

var (
	_ endpoint.Failer = CreateTaskResponse{}
	_ endpoint.Failer = TasksResponse{}
	_ endpoint.Failer = TaskResponse{}
	_ endpoint.Failer = UpdateTaskResponse{}
	_ endpoint.Failer = DeleteTaskResponse{}
)

type CreateTaskRequest struct {
	Title       string
	Description string
}

type CreateTaskResponse struct {
	Task tasksvc.Task `json:"task"`
	Err  error        `json:"-"`
}

func (r CreateTaskResponse) Failed() error { return r.Err }

type TasksRequest struct{}

type TasksResponse struct {
	Tasks []tasksvc.Task `json:"tasks"`
	Err   error          `json:"-"`
}

func (r TasksResponse) Failed() error { return r.Err }

type TaskRequest struct {
	TaskID uint64
}

type TaskResponse struct {
	Task tasksvc.Task `json:"task"`
	Err  error        `json:"-"`
}

func (r TaskResponse) Failed() error { return r.Err }

type UpdateTaskRequest struct {
	TaskID      uint64
	Title       string
	Description string
	Done        bool `json:"done,string"`
}

type UpdateTaskResponse struct {
	Task tasksvc.Task `json:"task"`
	Err  error        `json:"-"`
}

func (r UpdateTaskResponse) Failed() error { return r.Err }

type DeleteTaskRequest struct {
	TaskID uint64
}

type DeleteTaskResponse struct {
	Result bool  `json:"result"`
	Err    error `json:"-"`
}

func (r DeleteTaskResponse) Failed() error { return r.Err }
