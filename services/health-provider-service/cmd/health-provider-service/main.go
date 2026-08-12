package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/adapters/primary/grpc/handlers"
	authsvc "github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/adapters/secondary/external/auth"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/health-provider-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	grpcserver "google.golang.org/grpc"
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
	grpcAddr := grpcListenAddr(env.GetString("HEALTH_PROVIDER_SERVICE_GRPC_ADDR", "health-provider-service:9094"), "9094")

	tracerCfg := tracing.Config{
		ServiceName:    "health-provider-service",
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

	dbURL := env.GetString("DATABASE_URL", "")
	if err := db.InitDB(context.Background(), dbURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	pool := db.GetDB()
	defer pool.Close()

	authRepo, err := authsvc.NewRepository(env.GetString("AUTH_SERVICE_GRPC_ADDR", "auth-service:9092"))
	if err != nil {
		log.Fatalf("Failed to create auth repository: %v", err)
	}
	defer authRepo.Close()

	hospitalRepo := postgres.NewHospitalRepository(pool)
	svc := services.NewProviderService(hospitalRepo, authRepo)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	server := grpcserver.NewServer()
	grpchandlers.NewHealthProviderGRPCHandler(server, svc)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
		server.GracefulStop()
	}()

	log.Printf("Health provider service gRPC listening on %s", grpcAddr)
	if err := server.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}
