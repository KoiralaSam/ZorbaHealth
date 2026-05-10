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
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

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
	grpcServer := grpcserver.NewServer(grpcserver.UnaryInterceptor(grpcinterceptors.InternalAuthInterceptor))
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
		Handler:           mux,
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
