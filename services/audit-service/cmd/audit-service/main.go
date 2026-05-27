package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/adapters/primary/grpc/handlers"
	grpcinterceptors "github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/adapters/primary/grpc/interceptors"
	postgresrepo "github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/audit-service/internal/core/services"
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
	if err != nil || port == "" {
		return ":" + defaultPort
	}
	return ":" + port
}

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "audit-service",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	shutdownTracer, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := shutdownTracer(ctx); err != nil {
			log.Printf("tracer shutdown: %v", err)
		}
	}()
	defer cancel()

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	if err := db.InitDB(ctx, env.GetString("DATABASE_URL", "")); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	pool := db.GetDB()
	defer pool.Close()

	repo := postgresrepo.NewRepository(pool)
	svc := services.New(repo)

	grpcAddr := grpcListenAddr(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"), "50058")
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}
	defer lis.Close()

	serverOptions := append(
		tracing.WithTracingInterceptors(),
		grpcinterceptors.Chain(),
	)
	grpcSrv := grpcserver.NewServer(serverOptions...)
	grpchandlers.NewHandler(grpcSrv, svc)

	go func() {
		<-ctx.Done()
		grpcSrv.GracefulStop()
	}()

	log.Printf("audit-service listening on %s", grpcAddr)
	log.Fatal(grpcSrv.Serve(lis))
}
