package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	grpcadapter "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/primary/grpc/handlers"
	grpcinterceptors "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/primary/grpc/interceptors"
	openaiadapter "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/openai"
	postgresrepo "github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/adapters/secondary/repositories/postgres"
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
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

var grpcAddr = grpcListenAddr(env.GetString("MEDICAL_RECORDS_SERVICE_GRPC_ADDR", "health-records-service:50054"), "50054")

func main() {
	// --- Shutdown: cancel context on SIGINT/SIGTERM ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		cancel()
	}()

	// --- Database ---
	dbURL := env.GetString("DATABASE_URL", "")
	if err := db.InitDB(ctx, dbURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	pool := db.GetDB()
	defer pool.Close()

	// --- Outbound adapters ---
	store := postgresrepo.NewRepository(pool)
	openAIKey := os.Getenv("OPENAI_API_KEY")
	embedder := openaiadapter.NewClient(openAIKey)
	summarizer := openaiadapter.NewSummarizerClient(openAIKey)

	// --- Core service ---
	svc := services.NewHealthRecordsService(embedder, store, summarizer)

	// --- gRPC listener ---
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServer := grpcserver.NewServer(grpcserver.UnaryInterceptor(grpcinterceptors.Chain()))
	grpcadapter.NewGRPCHandler(grpcServer, svc, pool)

	log.Printf("health-records-service gRPC server listening on %s", grpcAddr)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Failed to serve gRPC server: %v", err)
			cancel()
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down gRPC server health-records-service")
	grpcServer.GracefulStop()
}
