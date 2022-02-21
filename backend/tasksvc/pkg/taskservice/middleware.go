package taskservice

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authendpoint"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
)

type Middleware func(Service) Service

func LoggingMiddleware(logger log.Logger) Middleware {
	return func(next Service) Service {
		return loggingMiddleware{logger, next}
	}
}

type loggingMiddleware struct {
	logger log.Logger
	next   Service
}

func (mw loggingMiddleware) CreateTask(ctx context.Context, a tasksvc.Auth, title, description string) (t tasksvc.Task, err error) {
	defer func() {
		mw.logger.Log(
			"method", "CreateTask",
			"access_uuid", a.AccessUUID,
			"title", title,
			"description", description,
			"user_id", a.UserID,
			"err", err,
		)
	}()
	return mw.next.CreateTask(ctx, a, title, description)
}

func (mw loggingMiddleware) Tasks(ctx context.Context, a tasksvc.Auth) (t []tasksvc.Task, err error) {
	defer func() {
		mw.logger.Log(
			"method", "Tasks",
			"access_uuid", a.AccessUUID,
			"user_id", a.UserID,
			"err", err,
		)
	}()
	return mw.next.Tasks(ctx, a)
}

func (mw loggingMiddleware) Task(ctx context.Context, a tasksvc.Auth, taskID uint64) (t tasksvc.Task, err error) {
	defer func() {
		mw.logger.Log(
			"method", "Task",
			"access_uuid", a.AccessUUID,
			"user_id", a.UserID,
			"task_id", taskID,
			"err", err,
		)
	}()
	return mw.next.Task(ctx, a, taskID)
}

func (mw loggingMiddleware) UpdateTask(ctx context.Context, a tasksvc.Auth, task tasksvc.Task) (t tasksvc.Task, err error) {
	defer func() {
		mw.logger.Log(
			"method", "UpdateTask",
			"access_uuid", a.AccessUUID,
			"user_id", a.UserID,
			"task_id", task.ID,
			"title", task.Title,
			"description", task.Description,
			"done", task.Done,
			"err", err,
		)
	}()
	return mw.next.UpdateTask(ctx, a, task)
}

func (mw loggingMiddleware) DeleteTask(ctx context.Context, a tasksvc.Auth, taskID uint64) (result bool, err error) {
	defer func() {
		mw.logger.Log(
			"method", "DeleteTask",
			"access_uuid", a.AccessUUID,
			"user_id", a.UserID,
			"task_id", taskID,
			"result", result,
			"err", err,
		)
	}()
	return mw.next.DeleteTask(ctx, a, taskID)
}

func InstrumentingMiddleware(counter metrics.Counter, latency metrics.Histogram, s Service) Middleware {
	return func(next Service) Service {
		return instrumentingMiddleware{counter, latency, next}
	}
}

type instrumentingMiddleware struct {
	requestCount   metrics.Counter
	requestLatency metrics.Histogram
	next           Service
}

func (mw instrumentingMiddleware) CreateTask(ctx context.Context, a tasksvc.Auth, title, description string) (t tasksvc.Task, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "create_task").Add(1)
		mw.requestLatency.With("method", "create_task").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.CreateTask(ctx, a, title, description)
}

func (mw instrumentingMiddleware) Tasks(ctx context.Context, a tasksvc.Auth) (t []tasksvc.Task, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "tasks").Add(1)
		mw.requestLatency.With("method", "tasks").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Tasks(ctx, a)
}

func (mw instrumentingMiddleware) Task(ctx context.Context, a tasksvc.Auth, taskID uint64) (t tasksvc.Task, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "task").Add(1)
		mw.requestLatency.With("method", "task").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.Task(ctx, a, taskID)
}

func (mw instrumentingMiddleware) UpdateTask(ctx context.Context, a tasksvc.Auth, task tasksvc.Task) (t tasksvc.Task, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "update_task").Add(1)
		mw.requestLatency.With("method", "update_task").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.UpdateTask(ctx, a, task)
}

func (mw instrumentingMiddleware) DeleteTask(ctx context.Context, a tasksvc.Auth, taskID uint64) (result bool, err error) {
	defer func(begin time.Time) {
		mw.requestCount.With("method", "delete_task").Add(1)
		mw.requestLatency.With("method", "delete_task").Observe(time.Since(begin).Seconds())
	}(time.Now())

	return mw.next.DeleteTask(ctx, a, taskID)
}

func ProxingMiddleware(ctx context.Context, validateUUID, isUserExists endpoint.Endpoint) Middleware {
	return func(next Service) Service {
		return proxingMiddleware{next, validateUUID, isUserExists}
	}
}

type proxingMiddleware struct {
	next         Service
	validateUUID endpoint.Endpoint
	isUserExists endpoint.Endpoint
}

func (mw proxingMiddleware) CreateTask(ctx context.Context, a tasksvc.Auth, title, description string) (tasksvc.Task, error) {
	err := mw.validate(ctx, a)
	if err != nil {
		return tasksvc.Task{}, err
	}

	return mw.next.CreateTask(ctx, a, title, description)
}

func (mw proxingMiddleware) Tasks(ctx context.Context, a tasksvc.Auth) ([]tasksvc.Task, error) {
	err := mw.validate(ctx, a)
	if err != nil {
		return nil, err
	}

	return mw.next.Tasks(ctx, a)
}

func (mw proxingMiddleware) Task(ctx context.Context, a tasksvc.Auth, taskID uint64) (tasksvc.Task, error) {
	err := mw.validate(ctx, a)
	if err != nil {
		return tasksvc.Task{}, err
	}

	return mw.next.Task(ctx, a, taskID)
}

func (mw proxingMiddleware) UpdateTask(ctx context.Context, a tasksvc.Auth, task tasksvc.Task) (tasksvc.Task, error) {
	err := mw.validate(ctx, a)
	if err != nil {
		return tasksvc.Task{}, err
	}

	return mw.next.UpdateTask(ctx, a, task)
}

func (mw proxingMiddleware) DeleteTask(ctx context.Context, a tasksvc.Auth, taskID uint64) (bool, error) {
	err := mw.validate(ctx, a)
	if err != nil {
		return false, err
	}

	return mw.next.DeleteTask(ctx, a, taskID)
}

func (mw proxingMiddleware) validate(ctx context.Context, a tasksvc.Auth) error {
	{
		response, err := mw.validateUUID(ctx, authendpoint.ValidateRequest{AccessUUID: a.AccessUUID})
		if err != nil {
			return err
		}

		resp := response.(authendpoint.ValidateResponse)
		if resp.Err != nil {
			return resp.Err
		}
	}
	{
		response, err := mw.isUserExists(ctx, userendpoint.IsExistsRequest{ID: a.UserID})
		if err != nil {
			return err
		}

		resp := response.(userendpoint.IsExistsResponse)
		if resp.Err != nil {
			return resp.Err
		}
	}
	return nil
}
