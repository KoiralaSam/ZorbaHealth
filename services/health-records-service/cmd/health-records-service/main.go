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
	"github.com/KoiralaSam/ZorbaHealth/services/health-records-service/internal/rag"
	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
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

var grpcAddr = grpcListenAddr(env.GetString("MEDICAL_RECORDS_SERVICE_GRPC_ADDR", "health-records-service:50054"), "50054")

func main() {
	tracerCfg := tracing.Config{
		ServiceName:    "health-records-service",
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

	// --- Shutdown: cancel context on SIGINT/SIGTERM ---
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
	embeddingProviderName := env.GetString("EMBEDDING_PROVIDER", "openai")
	llmProviderName := env.GetString("LLM_PROVIDER", "openai")

	embedder, err := openaiadapter.NewEmbeddingProvider(embeddingProviderName, openAIKey)
	if err != nil {
		log.Fatalf("embedding provider: %v", err)
	}
	summarizer, err := openaiadapter.NewLLMProvider(llmProviderName, openAIKey)
	if err != nil {
		log.Fatalf("llm provider: %v", err)
	}
	log.Printf(
		"health-records providers configured: embeddings=%s/%s llm=%s/%s",
		embedder.ProviderName(),
		embedder.ModelName(),
		summarizer.ProviderName(),
		summarizer.ModelName(),
	)

	auditConn, err := grpcclient.Dial(env.GetString("AUDIT_SERVICE_GRPC_ADDR", "audit-service:50058"))
	if err != nil {
		log.Fatalf("audit-service dial: %v", err)
	}
	defer auditConn.Close()
	auditClient := auditpb.NewAuditServiceClient(auditConn)
	ragPipeline := rag.NewPipeline(
		store,
		embedder,
		summarizer,
		rag.NewGRPCConsentChecker(auditClient),
		rag.NewGRPCAuditAdapter(auditClient, "health-records-service"),
		embedder.ModelName(),
	)

	// --- Core service ---
	svc := services.NewHealthRecordsService(embedder, store, summarizer, ragPipeline)

	// --- gRPC listener ---
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", grpcAddr, err)
	}

	grpcServerOptions := append(
		tracing.WithTracingInterceptors(),
		grpcserver.UnaryInterceptor(grpcinterceptors.Chain()),
	)
	grpcServer := grpcserver.NewServer(grpcServerOptions...)
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
