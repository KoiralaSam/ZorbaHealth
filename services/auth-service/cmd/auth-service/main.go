package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpchandler "github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/primary/grpc/handlers"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/primary/rmqconsumer"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/auth-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
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

var grpcAddr = grpcListenAddr(env.GetString("AUTH_SERVICE_GRPC_ADDR", "auth-service:9092"), "9092")

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "auth-service",
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

	userRepo := postgres.NewUserRepository(pool)
	authRepo := postgres.NewAuthRepository(pool)
	svc := services.NewAuthService(userRepo, authRepo)

	//connecting to rabbitmq
	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"), events.PatientExchange, events.AuthServicePatientQueueBindings)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer rabbitmq.Close()
	log.Println("Starting RabbitMQ connection")

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpcserver.NewServer(tracing.WithTracingInterceptors()...)
	grpchandler.NewAuthGRPCHandler(grpcServer, svc)

	log.Printf("Auth service gRPC server listening on %s (Login, RegisterPatient, RegisterHealthProvider, VerifyToken, Logout)", grpcAddr)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("gRPC server error: %v", err)
			cancel()
		}
	}()

	patientConsumer := rmqconsumer.NewPatientConsumer(rabbitmq)
	go func() {
		if err := patientConsumer.Listen(); err != nil {
			log.Printf("Failed to listen for patient messages: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down auth service gRPC server")
	grpcServer.GracefulStop()
}
