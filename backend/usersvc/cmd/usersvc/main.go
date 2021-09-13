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
	consulsd "github.com/go-kit/kit/sd/consul"
	kitgrpc "github.com/go-kit/kit/transport/grpc"
	"github.com/hashicorp/consul/api"
	"github.com/ichigozero/gtdkit/backend/usersvc/pb"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userendpoint"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/userservice"
	"github.com/ichigozero/gtdkit/backend/usersvc/pkg/usertransport"
	"github.com/oklog/oklog/pkg/group"
	"github.com/twinj/uuid"
	"google.golang.org/grpc"
)

func main() {
	fs := flag.NewFlagSet("usersvc", flag.ExitOnError)
	var (
		grpcAddr   = fs.String("grpc.addr", ":8080", "gRPC listen address")
		consulAddr = fs.String("consul.addr", "", "Consul agent address")
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
		service    = userservice.New(logger)
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

		_, port, _ := net.SplitHostPort(*grpcAddr)
		p, _ := strconv.Atoi(port)
		asr := &api.AgentServiceRegistration{
			ID:      uuid.NewV4().String(),
			Name:    "usersvc",
			Address: "localhost",
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
