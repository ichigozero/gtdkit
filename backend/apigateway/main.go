package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	consulsd "github.com/go-kit/kit/sd/consul"
	"github.com/gorilla/mux"
	"github.com/hashicorp/consul/api"
	authclient "github.com/ichigozero/gtdkit/backend/authsvc/client"
	"github.com/ichigozero/gtdkit/backend/authsvc/inmem"
	"github.com/ichigozero/gtdkit/backend/authsvc/pkg/authtransport"
	taskclient "github.com/ichigozero/gtdkit/backend/tasksvc/client"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/tasktransport"
)

func main() {
	var (
		httpAddr     = flag.String("http.addr", ":8000", "Address for HTTP (JSON) server")
		consulAddr   = flag.String("consul.addr", "", "Consul agent address")
		retryMax     = flag.Int("retry.max", 3, "per-request retries to different instances")
		retryTimeout = flag.Duration("retry.timeout", 500*time.Millisecond, "per-request timeout, including retries")
	)
	flag.Parse()

	var logger log.Logger
	{
		logger = log.NewLogfmtLogger(os.Stderr)
		logger = log.With(logger, "ts", log.DefaultTimestampUTC)
		logger = log.With(logger, "caller", log.DefaultCaller)
	}

	var (
		client      consulsd.Client
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

		client = consulsd.NewClient(consulClient)
		inmemClient = inmem.NewClient(consulClient)
	}

	r := mux.NewRouter()
	{
		endpoints, _ := authclient.New(client, logger, *retryMax, *retryTimeout)
		authHTTPHandler := authtransport.NewHTTPHandler(endpoints, inmemClient, logger)
		r.PathPrefix("/auth/v1").Handler(http.StripPrefix("/auth/v1", authHTTPHandler))
	}
	{
		endpoints, _ := taskclient.New(client, logger, *retryMax, *retryTimeout)
		taskHTTPHandler := tasktransport.NewHTTPHandler(endpoints, logger)
		r.PathPrefix("/task/v1").Handler(http.StripPrefix("/task/v1", taskHTTPHandler))
	}

	// Interrupt handler.
	errc := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// HTTP transport.
	go func() {
		logger.Log("transport", "HTTP", "addr", *httpAddr)
		errc <- http.ListenAndServe(*httpAddr, r)
	}()

	// Run!
	logger.Log("exit", <-errc)
}
