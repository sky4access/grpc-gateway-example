package main

import (
	"fmt"
	"github.com/namsral/flag"
	"os"
	"google.golang.org/grpc"
	pb "github.com/sky4access/grpc-gateway-example/pkg/greeter"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/sirupsen/logrus"

	"net"


	"net/http"
	"context"
	"golang.org/x/sync/errgroup"
	"github.com/pkg/errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/sky4access/grpc-gateway-example/cmd/greeter/internal/rpc"
)


var (
	// Version of the application.
	Version string
	// Build in in which the application was created.
	Build string
)


func main()  {

	// Define inputs.
	var in struct {
		grpcPort          int
		httpPort          int

		// agent contains configurations needed to talk to agents sitting
		// on sites.
		agent struct {
			grpcPort int

			insecure bool

			caFile   string
			certFile string
			keyFile  string
		}

		// control contains inputs that control the flow of the program.
		control struct {
			showVersion bool
		}

		db struct {
			password string
			server   string
			user     string
			dbName   string
		}
	}
	flag.IntVar(&in.grpcPort, "grpc-port", 50051, "port to listen for grpc traffic")
	flag.IntVar(&in.httpPort, "http-port", 8080, "port to listen for http traffic (grpc gateway)")
	flag.IntVar(&in.agent.grpcPort, "agent-grpc-port", 50051, "port to dial agent on")
	flag.BoolVar(&in.agent.insecure, "agent-insecure", false, "forget tls, who needs it?")
	flag.StringVar(&in.agent.caFile, "agent-ca-file", "", "tls ca file for agent communication")
	flag.StringVar(&in.agent.certFile, "agent-cert-file", "", "tls cert file for agent communication")
	flag.StringVar(&in.agent.keyFile, "agent-key-file", "", "tls key file for agent communication")
	flag.BoolVar(&in.control.showVersion, "version", false, "print the version and exit")

	flag.Parse()

	if in.control.showVersion {
		fmt.Printf("version %s build %s\n", Version, Build)
		os.Exit(0)
	}

	// Setup the root logger.
	log := logrus.New()


	// Setup grpc server.
	logE := logrus.NewEntry(log)
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_logrus.UnaryServerInterceptor(logE),
		)),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_logrus.StreamServerInterceptor(logE),
		)),
	)

	svc := rpc.NewService()

	pb.RegisterGreeterServer(grpcServer, svc)

	grpcAddr := fmt.Sprintf(":%v", in.grpcPort)
	grpcListener, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.WithError(err).Fatal("unable to listen for grpc traffic")
	}


	// Setup grpc gateway.
	// TODO: Consider removing insecure.
	// However: This is not a huge issue as this traffic is on localhost only.
		gwyMux := runtime.NewServeMux()
	gwyOpts := []grpc.DialOption{grpc.WithInsecure()}
	if err := pb.RegisterGreeterHandlerFromEndpoint(
		context.Background(), gwyMux,
		fmt.Sprintf("localhost%s", grpcAddr), gwyOpts,
	); err != nil {
		log.WithError(err).Fatal("unable to register grpc gateway")
	}
	gwyServer := http.Server{
		Addr:    fmt.Sprintf(":%v", in.httpPort),
		Handler: gwyMux,
	}

	// Launch goroutines.
	var g errgroup.Group
	g.Go(func() error {
		return errors.Wrap(grpcServer.Serve(grpcListener), "grpc server")
	})
	g.Go(func() error {
		// Filter out the error returned on graceful shutdown.
		if err := gwyServer.ListenAndServe(); err != http.ErrServerClosed {
			return errors.Wrap(err, "grpc gateway server")
		}
		return nil
	})


	// Catch shutdown signals and attempt a graceful shutdown.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-sigs
		log.WithField("signal", s).Info("caught signal")

		log.Info("shutting down grpc server")
		grpcServer.GracefulStop()

		log.Info("shutting down grpc gateway")
		// Give http shutdown process 3 seconds.
		stopGwyCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		if err := gwyServer.Shutdown(stopGwyCtx); err != nil {
			log.WithError(err).Error("shutting down grpc gateway was not flawless")
		}
		defer cancel()
	}()

	log.Info("all goroutines launched")
	// Wait for all goroutines to stop.
	if err := g.Wait(); err != nil {
		log.WithError(err).Info("one of the goroutines reported an error")
	}
	log.Info("done")
}
