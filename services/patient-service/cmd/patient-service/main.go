package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/primary/grpc/handlers"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/primary/grpc/interceptors"
	authsvc "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/auth"
	livekitadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/external/livekit"
	rmqadapter "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/messaging/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/postgres"
	redisrepo "github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/adapters/secondary/repositories/redis"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
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

var (
	grpcAddr = grpcListenAddr(env.GetString("PATIENT_SERVICE_GRPC_ADDR", "patient-service:9093"), "9093")
)

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "patient-service",
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
	bridgedCallRepo, err := redisrepo.NewBridgedCallRepository()
	if err != nil {
		log.Fatalf("Failed to create bridged call repository: %v", err)
	}

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
	rabbitmq, err := messaging.NewRabbitMQ(env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/"), events.PatientExchange, events.PatientPublisherQueueBindings)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer rabbitmq.Close()
	log.Println("Starting RabbitMQ connection")
	patientPublisher := rmqadapter.NewPatientPublisher(rabbitmq)
	schedulingPublisher := rmqadapter.NewSchedulingPublisher(rabbitmq)
	meetingRepo := postgres.NewMeetingRepository(db)
	liveKitProvider := livekitadapter.NewClient()

	var auditClient auditpb.AuditServiceClient
	auditConn, err := grpcclient.Dial(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Printf("audit-service dial failed (voice audit disabled): %v", err)
	} else {
		defer auditConn.Close()
		auditClient = auditpb.NewAuditServiceClient(auditConn)
	}

	// --- Core service ---
	svc := services.NewPatientService(postgresRepo, authRepo, pendingRegRepo, patientPublisher, auditClient)
	schedulingSvc := services.NewSchedulingService(meetingRepo, postgresRepo, bridgedCallRepo, liveKitProvider, schedulingPublisher, auditClient)

	// --- gRPC server: register handlers and serve ---
	opts := tracing.WithTracingInterceptors()
	opts = append(opts, grpcserver.ChainUnaryInterceptor(interceptors.ForwardedTokenInterceptor))
	grpcServer := grpcserver.NewServer(opts...)
	grpc.NewGRPCHandler(grpcServer, svc, schedulingSvc)
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
