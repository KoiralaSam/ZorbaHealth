package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/config"
	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/adapters/primary/grpc/handlers"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/adapters/primary/grpc/interceptors"
	llamacpp "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/adapters/secondary/external/llamacpp"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

func grpcListenAddr(addr string, defaultPort string) string {
	if addr == "" {
		return ":" + defaultPort
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ":" + defaultPort
	}
	return ":" + port
}

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "translation-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	sh, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer sh(ctx)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	provider := llamacpp.NewClient(
		cfg.TranslationModelBaseURL,
		cfg.ModelTimeout,
		cfg.TranslationModelName,
		cfg.ModelMaxTokens,
	)

	svc := services.NewTranslationService(provider, cfg.MaxTextLength)
	handler := grpchandlers.NewTranslationHandler(svc)

	grpcServerOptions := append(
		tracing.WithTracingInterceptors(),
		interceptors.Chain(),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle: 5 * time.Minute,
			Time:              2 * time.Minute,
			Timeout:           20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: false,
		}),
		grpc.MaxRecvMsgSize(4*1024*1024),
		grpc.MaxSendMsgSize(4*1024*1024),
	)
	grpcServer := grpc.NewServer(grpcServerOptions...)

	pb.RegisterTranslationServiceServer(grpcServer, handler)

	if cfg.EnableGRPCReflection {
		reflection.Register(grpcServer)
		log.Println("gRPC reflection enabled")
	}

	grpcAddr := grpcListenAddr(cfg.TranslationServiceGRPCAddr, "50057")

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("listen %s: %v", grpcAddr, err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("translation-service listening on %s", grpcAddr)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down gracefully...")

	stopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-stopped:
		log.Println("all RPCs completed, exiting")
	case <-time.After(10 * time.Second):
		log.Println("graceful stop timed out, forcing stop")
		grpcServer.Stop()
	}
}
