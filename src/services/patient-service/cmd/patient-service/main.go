package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	rmqadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/messaging/rabbitmq"
	grpc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/primary/grpc/handlers"
	authsvc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/auth"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres"
	redisrepo "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/redis"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
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

var (
	grpcAddr = grpcListenAddr(env.GetString("PATIENT_SERVICE_GRPC_ADDR", "patient-service:9093"), "9093")
)

func main() {
	// --- Database ---
	dbURL := env.GetString("DATABASE_URL", "")
	if err := db.InitDB(context.Background(), dbURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	db := db.GetDB()
	defer db.Close()

	// --- Repositories ---
	authServiceGRPCAddr := env.GetString("AUTH_SERVICE_GRPC_ADDR", "")
	postgresRepo := postgres.NewPatientRepository(db)
	authRepo, err := authsvc.NewAuthRepository(authServiceGRPCAddr)
	if err != nil {
		log.Fatalf("Failed to create auth repository: %v", err)
	}
	if closer, ok := authRepo.(authsvc.AuthRepositoryWithClose); ok {
		defer closer.Close()
	}

	// Redis store for pending registrations (until email verification).
	pendingRegRepo, err := redisrepo.NewPendingRegistrationRepository()
	if err != nil {
		log.Fatalf("Failed to create pending registration repository: %v", err)
	}

	// --- Core service ---
	svc := services.NewPatientService(postgresRepo, authRepo, pendingRegRepo)

	// --- Shutdown: cancel context on SIGINT/SIGTERM ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	// --- gRPC listener ---
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	// --- RabbitMQ: publish patient-registered events ---
	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer rabbitmq.Close()
	log.Println("Starting RabbitMQ connection")
	patientPublisher := rmqadapter.NewPatientPublisher(rabbitmq)

	// --- gRPC server: register handlers and serve ---
	grpcServer := grpcserver.NewServer()
	grpc.NewGRPCHandler(grpcServer, svc, patientPublisher)
	log.Printf("Starting gRPC server patient service on port %s", grpcAddr)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve gRPC server: %v", err)
			cancel()
		}
	}()

	// Block until shutdown; then graceful stop.
	<-ctx.Done()
	log.Println("Shutting down gRPC server patient service")
	grpcServer.GracefulStop()
}
