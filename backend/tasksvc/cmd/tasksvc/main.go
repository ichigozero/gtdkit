package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/go-kit/kit/log"
	consulsd "github.com/go-kit/kit/sd/consul"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/hashicorp/consul/api"
	authclient "github.com/ichigozero/gtdkit/backend/authsvc/client"
	"github.com/ichigozero/gtdkit/backend/tasksvc"
	"github.com/ichigozero/gtdkit/backend/tasksvc/db/gorm"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pb"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskendpoint"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/taskservice"
	"github.com/ichigozero/gtdkit/backend/tasksvc/pkg/tasktransport"
	userclient "github.com/ichigozero/gtdkit/backend/usersvc/client"
	"github.com/oklog/oklog/pkg/group"
	"github.com/twinj/uuid"
	"google.golang.org/grpc"
	"gorm.io/driver/sqlite"
	"gorm.io/driver/postgres"
	libgorm "gorm.io/gorm"
)

func main() {
	fs := flag.NewFlagSet("tasksvc", flag.ExitOnError)
	var (
		grpcAddr = fs.String(
			"grpc.addr",
			getEnv("GRPC_ADDR", ":8082"),
			"gRPC listen address",
		)
		consulAddr = fs.String(
			"consul.addr",
			getEnv("CONSUL_ADDR", ""),
			"Consul agent address",
		)
		databaseURL = fs.String(
			"database.url",
			getEnv("DATABASE_URL", ""),
			"Database URL",
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


	var db *libgorm.DB
	var err error
	{
		if *databaseURL != "" {
			db, err = libgorm.Open(postgres.Open(*databaseURL), &libgorm.Config{})
		} else {
			db, err = libgorm.Open(sqlite.Open("gorm.db"), &libgorm.Config{})
		}
		if err != nil {
			logger.Log("err", err)
			os.Exit(1)
		}
	}

	var (
		client    consulsd.Client
		registrar *consulsd.Registrar
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

		host, port, err := net.SplitHostPort(*grpcAddr)
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
			Name:    "tasksvc",
			Address: host,
			Port:    p,
		}

		client = consulsd.NewClient(consulClient)
		registrar = consulsd.NewRegistrar(client, asr, logger)
		registrar.Register()
		defer registrar.Deregister()
	}

	db.AutoMigrate(&tasksvc.Task{})
	taskRepository := gorm.NewTaskRepository(db)
	authEndpoints, _ := authclient.New(client, logger, *retryMax, *retryTimeout)
	userEndpoints, _ := userclient.New(client, logger, *retryMax, *retryTimeout)

	var service taskservice.Service
	{
		service = taskservice.New(taskRepository, logger)
		service = taskservice.ProxingMiddleware(
			context.Background(),
			authEndpoints.ValidateEndpoint,
			userEndpoints.IsExistsEndpoint,
		)(service)
	}

	var (
		endpoints  = taskendpoint.New(service, logger)
		grpcServer = tasktransport.NewGRPCServer(endpoints, logger)
	)

	var g group.Group
	{
		// The gRPC listener mounts the Go kit gRPC server we created.
		grpcListener, err := net.Listen("tcp", *grpcAddr)
		if err != nil {
			logger.Log("transport", "gRPC", "during", "Listen", "err", err)
			registrar.Deregister()
			os.Exit(1)
		}
		g.Add(func() error {
			logger.Log("transport", "gRPC", "addr", *grpcAddr)
			baseServer := grpc.NewServer(grpc.UnaryInterceptor(kitgrpc.Interceptor))
			pb.RegisterTaskSVCServer(baseServer, grpcServer)
			return baseServer.Serve(grpcListener)
		}, func(error) {
			grpcListener.Close()
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
