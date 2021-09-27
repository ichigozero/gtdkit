package authtransport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/transport"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/ichigozero/gtdkit/backend/authsvc"
	"github.com/ichigozero/gtdkit/backend/authsvc/inmem"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authendpoint"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authservice"
	"github.com/ichigozero/gtdkit/backend/usersvc"
)

func NewHTTPHandler(endpoints authendpoint.Set, client inmem.Client, logger log.Logger) http.Handler {
	options := []httptransport.ServerOption{
		httptransport.ServerErrorEncoder(errorEncoder),
		httptransport.ServerErrorHandler(transport.NewLogErrorHandler(logger)),
	}

	m := http.NewServeMux()

	loginHandler := httptransport.NewServer(
		endpoints.LoginEndpoint,
		decodeHTTPLoginRequest,
		encodeHTTPGenericResponse,
		options...,
	)

	var logoutEndpoint endpoint.Endpoint
	{
		kf := func(token *stdjwt.Token) (interface{}, error) {
			return []byte(os.Getenv("ACCESS_SECRET")), nil
		}

		logoutEndpoint = endpoints.LogoutEndpoint
		logoutEndpoint = NewAuthenticater(client)(logoutEndpoint)
		logoutEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(logoutEndpoint)
	}

	logoutHandler := httptransport.NewServer(
		logoutEndpoint,
		decodeHTTPLogoutRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	var refreshEndpoint endpoint.Endpoint
	{
		kf := func(token *stdjwt.Token) (interface{}, error) {
			return []byte(os.Getenv("REFRESH_SECRET")), nil
		}

		refreshEndpoint = endpoints.RefreshEndpoint
		refreshEndpoint = kitjwt.NewParser(
			kf,
			stdjwt.SigningMethodHS256,
			kitjwt.MapClaimsFactory,
		)(refreshEndpoint)
	}

	refreshHandler := httptransport.NewServer(
		refreshEndpoint,
		decodeHTTPRefreshRequest,
		encodeHTTPGenericResponse,
		append(options, httptransport.ServerBefore(kitjwt.HTTPToContext()))...,
	)

	m.Handle("/login", loginHandler)
	m.Handle("/logout", logoutHandler)
	m.Handle("/refresh", refreshHandler)

	return m
}

func NewHTTPClient(instance string, logger log.Logger) (authservice.Service, error) {
	// Quickly sanitize the instance string.
	if !strings.HasPrefix(instance, "http") {
		instance = "http://" + instance
	}
	u, err := url.Parse(instance)
	if err != nil {
		return nil, err
	}

	var options []httptransport.ClientOption

	var loginEndpoint endpoint.Endpoint
	{
		loginEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/login"),
			encodeHTTPGenericRequest,
			decodeHTTPLoginResponse,
			options...,
		).Endpoint()
	}

	var logoutEndpoint endpoint.Endpoint
	{
		logoutEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/logout"),
			encodeHTTPGenericRequest,
			decodeHTTPLogoutResponse,
			append(options, httptransport.ClientBefore(kitjwt.ContextToHTTP()))...,
		).Endpoint()
	}

	var refreshEndpoint endpoint.Endpoint
	{
		refreshEndpoint = httptransport.NewClient(
			"POST",
			copyURL(u, "/refresh"),
			encodeHTTPGenericRequest,
			decodeHTTPRefreshResponse,
			append(options, httptransport.ClientBefore(kitjwt.ContextToHTTP()))...,
		).Endpoint()
	}

	return authendpoint.Set{
		LoginEndpoint:   loginEndpoint,
		LogoutEndpoint:  logoutEndpoint,
		RefreshEndpoint: refreshEndpoint,
	}, nil
}

func copyURL(base *url.URL, path string) *url.URL {
	next := *base
	next.Path = path
	return &next
}

func errorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	w.WriteHeader(err2code(err))
	json.NewEncoder(w).Encode(errorWrapper{Error: err.Error()})
}

func err2code(err error) int {
	switch err {
	case usersvc.ErrInvalidArgument:
		return http.StatusBadRequest
	case usersvc.ErrUserNotFound:
		return http.StatusUnauthorized
	case authsvc.ErrInvalidArgument:
		return http.StatusBadRequest
	case authsvc.ErrUserIDContextMissing:
		return http.StatusUnauthorized
	case inmem.ErrKeyNotFound:
		return http.StatusUnauthorized
	}
	return http.StatusInternalServerError
}

type errorWrapper struct {
	Error string `json:"error"`
}

func decodeHTTPLoginRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req authendpoint.LoginRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func decodeHTTPLoginResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp authendpoint.LoginResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

func decodeHTTPLogoutRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return authendpoint.LogoutRequest{}, nil
}

func decodeHTTPLogoutResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp authendpoint.LogoutResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

func decodeHTTPRefreshRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return authendpoint.RefreshRequest{}, nil
}

func decodeHTTPRefreshResponse(_ context.Context, r *http.Response) (interface{}, error) {
	if r.StatusCode != http.StatusOK {
		return nil, errors.New(r.Status)
	}
	var resp authendpoint.RefreshResponse
	err := json.NewDecoder(r.Body).Decode(&resp)
	return resp, err
}

// encodeHTTPGenericRequest is a transport/http.EncodeRequestFunc that
// JSON-encodes any request to the request body. Primarily useful in a client.
func encodeHTTPGenericRequest(_ context.Context, r *http.Request, request interface{}) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(request); err != nil {
		return err
	}
	r.Body = ioutil.NopCloser(&buf)
	return nil
}

// encodeHTTPGenericResponse is a transport/http.EncodeResponseFunc that encodes
// the response as JSON to the response writer. Primarily useful in a server.
func encodeHTTPGenericResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	if f, ok := response.(endpoint.Failer); ok && f.Failed() != nil {
		errorEncoder(ctx, f.Failed(), w)
		return nil
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}
