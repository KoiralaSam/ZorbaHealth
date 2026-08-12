package main

import (
	"context"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/primary/grpc/handlers"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/primary/grpc/interceptors"
	auditadapter "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/secondary/external/audit"
	livekitadapter "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/secondary/external/livekit"
	rmqadapter "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/secondary/messaging/rabbitmq"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/ports/outbound"
	"github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	sharedgrpcauth "github.com/KoiralaSam/ZorbaHealth/shared/grpc/auth"
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

func main() {
	grpcAddr := grpcListenAddr(env.GetString("APPOINTMENT_SERVICE_GRPC_ADDR", "appointment-service:9099"), "9099")

	tracerCfg := tracing.Config{
		ServiceName:    "appointment-service",
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

	slog.Info("starting appointment-service", "addr", grpcAddr)

	dbURL := env.GetString("DATABASE_URL", "")
	if err := db.InitDB(context.Background(), dbURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	pool := db.GetDB()
	defer pool.Close()

	var eventPub outbound.EventPublisher = noopPublisher{}
	rabbitURI := env.GetString("RABBITMQ_URL", "")
	if rabbitURI == "" {
		rabbitURI = env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	}
	rabbitmq, err := messaging.NewRabbitMQ(rabbitURI, events.PatientExchange, events.PatientPublisherQueueBindings)
	if err != nil {
		log.Printf("RabbitMQ unavailable (notifications disabled): %v", err)
	} else {
		defer rabbitmq.Close()
		eventPub = rmqadapter.NewAppointmentPublisher(rabbitmq)
	}

	var auditClient auditpb.AuditServiceClient
	auditConn, err := grpcclient.Dial(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Printf("audit-service dial failed (audit disabled): %v", err)
	} else {
		defer auditConn.Close()
		auditClient = auditpb.NewAuditServiceClient(auditConn)
	}
	auditLogger := auditadapter.NewLogger(auditClient)

	apptRepo := postgres.NewAppointmentRepository(pool)
	availRepo := postgres.NewAvailabilityRepository(pool)
	livekit := livekitadapter.NewClient()

	apptSvc := services.NewAppointmentService(apptRepo, availRepo, eventPub, livekit, auditLogger)
	availSvc := services.NewAvailabilityService(availRepo, auditLogger)

	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	opts := tracing.WithTracingInterceptors()
	opts = append(opts, grpcserver.ChainUnaryInterceptor(
		sharedgrpcauth.UnaryServerInterceptor(sharedgrpcauth.InternalServerConfig{
			SharedSecret: env.GetString("INTERNAL_SERVICE_SECRET", ""),
			AllowedServices: map[string]struct{}{
				"api-gateway": {},
				"mcp-server":  {},
			},
		}),
		interceptors.ForwardedTokenInterceptor,
	))
	grpcServer := grpcserver.NewServer(opts...)
	grpchandlers.NewAppointmentGRPCHandler(grpcServer, apptSvc, availSvc)

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
		grpcServer.GracefulStop()
	}()

	slog.Info("appointment-service gRPC listening", "addr", grpcAddr)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC serve error: %v", err)
	}
}

type noopPublisher struct{}

func (noopPublisher) PublishAppointmentBooked(context.Context, *events.AppointmentBookedData) error {
	return nil
}
func (noopPublisher) PublishAppointmentCancelled(context.Context, *events.AppointmentCancelledData) error {
	return nil
}
