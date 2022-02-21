package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"text/tabwriter"

	"github.com/go-kit/kit/log"
	kitprometheus "github.com/go-kit/kit/metrics/prometheus"
	consulsd "github.com/go-kit/kit/sd/consul"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/hashicorp/consul/api"
	"github.com/ichigozero/gtdkit/backend/usersvc"
	"github.com/ichigozero/gtdkit/backend/usersvc/db/gorm"
	"github.com/ichigozero/gtdkit/backend/usersvc/pb"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/usertransport"
	"github.com/oklog/oklog/pkg/group"
	stdprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/twinj/uuid"
	"google.golang.org/grpc"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	libgorm "gorm.io/gorm"
)

func main() {
	fs := flag.NewFlagSet("usersvc", flag.ExitOnError)
	var (
		grpcAddr    = fs.String("grpc.addr", getEnv("GRPC_ADDR", ":8080"), "gRPC listen address")
		consulAddr  = fs.String("consul.addr", getEnv("CONSUL_ADDR", ""), "Consul agent address")
		databaseURL = fs.String("database.url", getEnv("DATABASE_URL", ""), "Database URL")
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

	db.AutoMigrate(&usersvc.User{})
	userRepository := gorm.NewUserRepository(db)

	fieldKeys := []string{"method"}

	var service userservice.Service
	{
		service = userservice.New(userRepository, logger)
		service = userservice.InstrumentingMiddleware(
			kitprometheus.NewCounterFrom(stdprometheus.CounterOpts{
				Namespace: "api",
				Subsystem: "user_service",
				Name:      "request_count",
				Help:      "Number of requests received.",
			}, fieldKeys),
			kitprometheus.NewSummaryFrom(stdprometheus.SummaryOpts{
				Namespace: "api",
				Subsystem: "user_service",
				Name:      "request_latency_microseconds",
				Help:      "Total duration of requests in microseconds.",
			}, fieldKeys),
			service,
		)(service)
	}

	var (
		endpoints  = userendpoint.New(service, logger)
		grpcServer = usertransport.NewGRPCServer(endpoints, logger)
	)

	var registrar *consulsd.Registrar
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
			Name:    "usersvc",
			Address: host,
			Port:    p,
		}

		client := consulsd.NewClient(consulClient)
		registrar = consulsd.NewRegistrar(client, asr, logger)
		registrar.Register()
		defer registrar.Deregister()
	}

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
			pb.RegisterUserServer(baseServer, grpcServer)
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
