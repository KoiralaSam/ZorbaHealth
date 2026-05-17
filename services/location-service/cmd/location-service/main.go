package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	grpchandlers "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/grpc/handlers"
	grpcinterceptors "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/grpc/interceptors"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/http/auth"
	httphandlers "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/http/handlers"
	rmqconsumer "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/primary/rabbitmq"
	geolocation "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/secondary/geolocation"
	memoryrepo "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/secondary/repositories/memory"
	redisrepo "github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/secondary/repositories/redis"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/adapters/secondary/stub"
	"github.com/KoiralaSam/ZorbaHealth/services/location-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/events"
	"github.com/KoiralaSam/ZorbaHealth/shared/messaging"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

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
	tracerCfg := tracing.Config{
		ServiceName:    "location-service",
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
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	// ── Configuration / Infrastructure ─────────────────────────────────────
	httpAddr := env.GetString("LOCATION_SERVICE_HTTP_ADDR", ":8090")
	grpcAddr := grpcListenAddr(env.GetString("LOCATION_SERVICE_GRPC_ADDR", "location-service:50051"), "50051")
	rabbitMQURI := env.GetString("RABBITMQ_URI", "amqp://guest:guest@rabbitmq:5672/")
	jwtSecret := env.GetString("PATIENT_SERVICE_JWT_SECRET", "")

	// ── Secondary adapters (outbound ports) ────────────────────────────────
	locationRepo, err := redisrepo.NewLocationRepository()
	if err != nil {
		log.Fatalf("redis location repository: %v", err)
	}
	registry := memoryrepo.NewInMemoryConnectionRegistry()

	geo, err := geolocation.NewIPAPIProvider()
	if err != nil {
		log.Fatalf("ip geolocation: %v", err)
	}
	hospitals := stub.NewNoopHospitalFinder()

	// ── Core service ───────────────────────────────────────────────────────
	svc := services.NewLocationService(locationRepo, registry, geo, hospitals)

	// ── Primary adapters (inbound adapters) ────────────────────────────────

	// 1) RabbitMQ consumer — receives call lifecycle events and pushes WS commands.
	rmq, err := messaging.NewRabbitMQ(rabbitMQURI, events.CallsExchange, events.LocationServiceCallsQueueBindings)
	if err != nil {
		log.Fatalf("rabbitmq: %v", err)
	}
	defer rmq.Close()

	callConsumer := rmqconsumer.NewCallEventConsumer(rmq, svc)
	go func() {
		if err := callConsumer.Listen(); err != nil {
			log.Printf("call event consumer: %v", err)
		}
	}()

	// 2) gRPC server — serves GetLocation / FindNearestHospital.
	grpcLis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("gRPC listen on %s: %v", grpcAddr, err)
	}
	grpcServerOptions := append(
		tracing.WithTracingInterceptors(),
		grpcserver.UnaryInterceptor(grpcinterceptors.InternalAuthInterceptor),
	)
	grpcServer := grpcserver.NewServer(grpcServerOptions...)
	grpchandlers.NewLocationGRPCHandler(grpcServer, svc)
	go func() {
		log.Printf("location-service gRPC listening on %s", grpcAddr)
		if err := grpcServer.Serve(grpcLis); err != nil {
			// grpcServer.Serve returns an error on GracefulStop; log only unexpected issues.
			log.Printf("gRPC serve: %v", err)
		}
	}()

	// 3) HTTP server — WebSocket endpoint.
	mux := http.NewServeMux()
	wsHandler := &httphandlers.WebSocketHandler{
		Service: svc,
		Auth:    auth.NewPatientJWTAuth(jwtSecret),
	}
	mux.HandleFunc("GET /ws/location", wsHandler.HandleConnect)

	server := &http.Server{
		Addr:              httpAddr,
		Handler:           otelhttp.NewHandler(mux, "location-service"),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("location-service HTTP listening on %s", httpAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down location-service")

	grpcServer.GracefulStop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown: %v", err)
	}
}
