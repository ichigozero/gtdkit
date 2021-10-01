package taskservice

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
)

type Service interface {
	CreateTask(ctx context.Context, a tasksvc.Auth, title, description string) (tasksvc.Task, error)
	Tasks(ctx context.Context, a tasksvc.Auth) ([]tasksvc.Task, error)
	Task(ctx context.Context, a tasksvc.Auth, taskID uint64) (tasksvc.Task, error)
	UpdateTask(ctx context.Context, a tasksvc.Auth, task tasksvc.Task) (tasksvc.Task, error)
	DeleteTask(ctx context.Context, a tasksvc.Auth, taskID uint64) (bool, error)
}

func New(t tasksvc.TaskRepository, logger log.Logger) Service {
	var svc Service
	{
		svc = NewBasicService(t)
		svc = LoggingMiddleware(logger)(svc)
	}
	return svc
}

type basicService struct {
	tasks tasksvc.TaskRepository
}

func NewBasicService(t tasksvc.TaskRepository) Service {
	return basicService{tasks: t}
}

func (s basicService) CreateTask(_ context.Context, a tasksvc.Auth, title, description string) (tasksvc.Task, error) {
	if title == "" || a.UserID == 0 {
		return tasksvc.Task{}, tasksvc.ErrInvalidArgument
	}
	return s.tasks.Create(title, description, a.UserID)
}

func (s basicService) Tasks(_ context.Context, a tasksvc.Auth) ([]tasksvc.Task, error) {
	if a.UserID == 0 {
		return nil, tasksvc.ErrInvalidArgument
	}
	return s.tasks.FindAll(a.UserID)
}

func (s basicService) Task(_ context.Context, a tasksvc.Auth, taskID uint64) (tasksvc.Task, error) {
	if a.UserID == 0 || taskID == 0 {
		return tasksvc.Task{}, tasksvc.ErrInvalidArgument
	}
	return s.tasks.Find(a.UserID, taskID)
}

func (s basicService) UpdateTask(_ context.Context, a tasksvc.Auth, task tasksvc.Task) (tasksvc.Task, error) {
	if a.UserID == 0 || task.ID == 0 {
		return tasksvc.Task{}, tasksvc.ErrInvalidArgument
	}
	return s.tasks.Update(task)
}

func (s basicService) DeleteTask(_ context.Context, a tasksvc.Auth, taskID uint64) (bool, error) {
	if a.UserID == 0 || taskID == 0 {
		return false, tasksvc.ErrInvalidArgument
	}
	return s.tasks.Delete(a.UserID, taskID)
}
