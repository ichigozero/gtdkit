package tasktransport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/mux"
	"github.com/ichigozero/gtdkit/backend/authsvc"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewHTTPHandler(endpoints taskendpoint.Set, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
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

	createTaskHandler := httptransport.NewServer(
		createTaskEndpoint,
		decodeHTTPCreateTaskRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	var tasksEndpoint endpoint.Endpoint
	{
		tasksEndpoint = endpoints.TasksEndpoint
		tasksEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(tasksEndpoint)
	}

	tasksHandler := httptransport.NewServer(
		tasksEndpoint,
		decodeHTTPTasksRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	var taskEndpoint endpoint.Endpoint
	{
		taskEndpoint = endpoints.TaskEndpoint
		taskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(taskEndpoint)
	}

	taskHandler := httptransport.NewServer(
		taskEndpoint,
		decodeHTTPTaskRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	var updateTaskEndpoint endpoint.Endpoint
	{
		updateTaskEndpoint = endpoints.UpdateTaskEndpoint
		updateTaskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(updateTaskEndpoint)
	}

	updateTaskHandler := httptransport.NewServer(
		updateTaskEndpoint,
		decodeHTTPUpdateTaskRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	var deleteTaskEndpoint endpoint.Endpoint
	{
		deleteTaskEndpoint = endpoints.DeleteTaskEndpoint
		deleteTaskEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(deleteTaskEndpoint)
	}

	deleteTaskHandler := httptransport.NewServer(
		deleteTaskEndpoint,
		decodeHTTPDeleteTaskRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	r := mux.NewRouter()

	r.Methods("POST").Path("/create").Handler(createTaskHandler)
	r.Methods("GET").Path("/tasks").Handler(tasksHandler)
	r.Methods("GET").Path("/task/{task_id}").Handler(taskHandler)
	r.Methods("PUT").Path("/task/{task_id}").Handler(updateTaskHandler)
	r.Methods("DELETE").Path("/task/{task_id}").Handler(deleteTaskHandler)
	r.Methods("GET").Path("/metrics").Handler(promhttp.Handler())

	return r
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

type errorWrapper struct {
	Error string `json:"error"`
}

func err2code(err error) int {
	switch err {
	case kitjwt.ErrTokenExpired, usersvc.ErrUserNotFound, authsvc.ErrUserIDContextMissing:
		return http.StatusUnauthorized
	case usersvc.ErrInvalidArgument, authsvc.ErrInvalidArgument, tasksvc.ErrInvalidArgument:
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func decodeHTTPCreateTaskRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req taskendpoint.CreateTaskRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func decodeHTTPTasksRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return taskendpoint.TasksRequest{}, nil
}

func decodeHTTPTaskRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	taskID, err := strconv.ParseUint(vars["task_id"], 10, 64)
	if err != nil {
		return nil, ErrBadRouting
	}

	return taskendpoint.TaskRequest{
		TaskID: taskID,
	}, nil
}

func decodeHTTPUpdateTaskRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	taskID, err := strconv.ParseUint(vars["task_id"], 10, 64)
	if err != nil {
		return nil, ErrBadRouting
	}

	var req taskendpoint.UpdateTaskRequest

	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, tasksvc.ErrInvalidArgument
	}

	req.TaskID = taskID

	return req, nil
}

func decodeHTTPDeleteTaskRequest(_ context.Context, r *http.Request) (interface{}, error) {
	vars := mux.Vars(r)
	taskID, err := strconv.ParseUint(vars["task_id"], 10, 64)
	if err != nil {
		return nil, ErrBadRouting
	}

	return taskendpoint.DeleteTaskRequest{
		TaskID: taskID,
	}, nil
}

// ErrBadRouting is returned when an expected path variable is missing.
// It always indicates programmer error.
var ErrBadRouting = errors.New("inconsistent mapping between route and handler (programmer error)")

func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
