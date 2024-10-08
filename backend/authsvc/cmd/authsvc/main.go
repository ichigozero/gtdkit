package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/ichigozero/gtdkit/backend/authsvc/inmem"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authendpoint"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authservice"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authtransport"
	userclient "github.com/ichigozero/gtdkit/backend/usersvc/client"
	"github.com/oklog/oklog/pkg/group"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/twinj/uuid"
)

func main() {
	fs := flag.NewFlagSet("authsvc", flag.ExitOnError)
	var (
		httpAddr = fs.String(
			"http.addr",
			getEnv("HTTP_ADDR", ":8081"),
			"HTTP listen address",
		)
		consulAddr = fs.String(
			"consul.addr",
			getEnv("CONSUL_ADDR", ""),
			"Consul agent address",
		)
		retryMax = flag.Int(
			"retry.max",
			getEnvAsInt("RETRY_MAX", 3),
			"per-request retries to different instances",
		)
		retryTimeout = flag.Duration(
			"retry.timeout",
			time.Duration(getEnvAsInt("RETRY_TIMEOUT", 500))*time.Millisecond,
			"per-request timeout, including retries",
		)
	)

	fs.Usage = usageFor(fs, os.Args[0]+" [flags]")
	fs.Parse(os.Args[1:])

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var (
		client      consulsd.Client
		registrar   *consulsd.Registrar
		inmemClient inmem.Client
	)
	{
		consulConfig := api.DefaultConfig()
		if len(*consulAddr) > 0 {
			consulConfig.Address = *consulAddr
		}
		consulClient, err := api.NewClient(consulConfig)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}

		host, port, err := net.SplitHostPort(*httpAddr)
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
		if host == "" {
			host = "localhost"
		}

		p, _ := strconv.Atoi(port)
		asr := &api.AgentServiceRegistration{
			ID:      uuid.NewV4().String(),
			Name:    "authsvc",
			Address: host,
			Port:    p,
		}

		client = consulsd.NewClient(consulClient)
		registrar = consulsd.NewRegistrar(client, asr, logger)
		registrar.Register()
		defer registrar.Deregister()

		inmemClient = inmem.NewClient(consulClient)
	}

	userEndpoints, _ := userclient.New(client, logger, *retryMax, *retryTimeout)

	fieldKeys := []string{"method"}

	var service authservice.Service
	{
		service = authservice.New(authservice.NewTokenizer(), inmemClient, logger)
		service = authservice.InstrumentingMiddleware(
			kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "api",
				Subsystem: "auth_service",
				Name:      "request_count",
				Help:      "Number of requests received.",
			}, fieldKeys),
			kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
				Namespace: "api",
				Subsystem: "auth_service",
				Name:      "request_latency_microseconds",
				Help:      "Total duration of requests in microseconds.",
			}, fieldKeys),
			service,
		)(service)
		service = authservice.ProxingMiddleware(
			context.Background(),
			userEndpoints.UserIDEndpoint,
			userEndpoints.IsExistsEndpoint,
		)(service)
	}

	var (
		endpoints   = authendpoint.New(service, logger)
		httpHandler = authtransport.NewHTTPHandler(endpoints, inmemClient, logger)
	)

	var g group.Group
	{
		// The HTTP listener mounts the Go kit HTTP handler we created.
		httpListener, err := net.Listen("tcp", *httpAddr)
		if err != nil {
			logger.Log("transport", "HTTP", "during", "Listen", "err", err)
			registrar.Deregister()
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "HTTP", "addr", *httpAddr)
			return http.Serve(httpListener, httpHandler)
		}, func(error) {
			httpListener.Close()
		})
	}
	{
		// This function just sits and waits for ctrl-C.
		cancelInterrupt := make(chan struct{})
		g.Add(func() error {
			c := make(chan os.Signal, 1)
			signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
			select {
			case sig := <-c:
				return fmt.Errorf("received signal %s", sig)
			case <-cancelInterrupt:
				return nil
			}
		}, func(error) {
			close(cancelInterrupt)
		})
	}
	logger.Log("exit", g.Run())
}

func usageFor(fs *flag.FlagSet, short string) func() {
	return func() {
		fmt.Fprintf(os.Stderr, "USAGE\n")
		fmt.Fprintf(os.Stderr, "  %s\n", short)
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "FLAGS\n")
		w := tabwriter.NewWriter(os.Stderr, 0, 2, 2, ' ', 0)
		fs.VisitAll(func(f *flag.Flag) {
			fmt.Fprintf(w, "\t-%s %s\t%s\n", f.Name, f.DefValue, f.Usage)
		})
		w.Flush()
		fmt.Fprintf(os.Stderr, "\n")
	}
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) int {
	value, exists := os.LookupEnv(key)
	if !exists {
		return fallback
	}

	if v, err := strconv.Atoi(value); err == nil {
		return v
	}
	return fallback
}
