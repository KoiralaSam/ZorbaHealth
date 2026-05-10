package main

import (
	"log"
	"net/http"
	"os"

	livekitprimary "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/primary/livekit"
	jwtissuer "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/auth/jwt"
	deepgramadapter "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/deepgram"
	elevenlabsadapter "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/elevenlabs"
	livekitsecondary "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/livekit"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/mcpclient"
	openaiadapter "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/openai"
	patientserviceadapter "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/external/patientservice"
	healthrecordsadapter "github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/adapters/secondary/healthrecords"
	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

func main() {
	healthAddr := env.GetString(
		"HEALTH_RECORDS_SERVICE_GRPC_ADDR",
		env.GetString("MEDICAL_RECORDS_SERVICE_GRPC_ADDR", "health-records-service:50054"),
	)
	patientAddr := env.GetString("PATIENT_SERVICE_GRPC_ADDR", "patient-service:9093")

	healthConn, err := grpcclient.Dial(healthAddr)
	if err != nil {
		log.Fatalf("health-records-service dial failed: %v", err)
	}
	defer func() {
		if err := healthConn.Close(); err != nil {
			log.Printf("health connection close error: %v", err)
		}
	}()

	patientClient, err := patientserviceadapter.NewClient(patientAddr)
	if err != nil {
		log.Fatalf("patient-service client init failed: %v", err)
	}
	defer func() {
		if err := patientClient.Close(); err != nil {
			log.Printf("patient client close error: %v", err)
		}
	}()

	mcpBinary := env.GetString("MCP_SERVER_BINARY", "/app/mcp-server")
	mcp, err := mcpclient.New(mcpBinary)
	if err != nil {
		log.Fatalf("mcp init failed: %v", err)
	}
	defer func() {
		if err := mcp.Close(); err != nil {
			log.Printf("mcp close error: %v", err)
		}
	}()

	healthClient := healthrecordsadapter.NewClient(healthpb.NewHealthRecordServiceClient(healthConn))
	workerService := services.NewAgentWorkerService(
		jwtissuer.NewSessionTokenIssuer(),
		patientClient,
		healthClient,
		deepgramadapter.NewDeepgram(os.Getenv("DEEPGRAM_API_KEY")),
		openaiadapter.NewOpenAI(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_MODEL")),
		elevenlabsadapter.NewElevenLabs(os.Getenv("ELEVENLABS_API_KEY")),
		mcp,
		livekitsecondary.NewRoomGateway(),
	)
	h := &livekitprimary.Handler{
		Service: workerService,
		Verifier: livekitsecondary.NewWebhookVerifier(
			os.Getenv("LIVEKIT_API_KEY"),
			os.Getenv("LIVEKIT_API_SECRET"),
		),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/livekit", h.HandleWebhook)
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	listenAddr := env.GetString("AGENT_WORKER_HTTP_ADDR", ":8090")
	log.Printf("agent-worker-service listening on %s", listenAddr)
	log.Fatal(http.ListenAndServe(listenAddr, mux))
}
