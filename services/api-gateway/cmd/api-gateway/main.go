package main

import (
	"context"
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	httpAddr = env.GetString("API_GATEWAY_HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	shutdown, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Printf("tracer shutdown: %v", err)
		}
	}()

	if dbURL := env.GetString("DATABASE_URL", ""); dbURL != "" {
		if err := db.InitDB(context.Background(), dbURL); err != nil {
			log.Printf("database unavailable; DB-backed routes disabled: %v", err)
		}
	}
	if db.GetDB() != nil {
		defer db.GetDB().Close()
	}

	mux := http.NewServeMux()

	// CORS preflight: OPTIONS must be handled for each path (browser sends OPTIONS before POST)
	optCORS := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify-otp", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/records/summary", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/profile", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/consents", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/records/answer", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/audit", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/incidents", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/patient/audit", optCORS)

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.Handle("POST /api/v1/auth/patient/login", tracing.WrapHandlerFunc(corsMiddleware(PatientLoginHandler), "/api/v1/auth/patient/login"))
	mux.Handle("POST /api/v1/auth/patient/register", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterHandler), "/api/v1/auth/patient/register"))
	mux.Handle("POST /api/v1/auth/patient/register/verify", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyHandler), "/api/v1/auth/patient/register/verify"))
	mux.Handle("POST /api/v1/auth/patient/register/verify-otp", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyOTPHandler), "/api/v1/auth/patient/register/verify-otp"))
	mux.Handle("POST /api/v1/auth/hospital/login", tracing.WrapHandlerFunc(corsMiddleware(HospitalLoginHandler), "/api/v1/auth/hospital/login"))
	mux.Handle("POST /api/v1/hospital/records/summary", tracing.WrapHandlerFunc(corsMiddleware(HospitalPatientSummaryHandler), "/api/v1/hospital/records/summary"))
	mux.Handle("GET /api/v1/patient/profile", tracing.WrapHandlerFunc(corsMiddleware(PatientProfileHandler), "/api/v1/patient/profile"))
	mux.Handle("GET /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientListConsentsHandler), "/api/v1/patient/consents"))
	mux.Handle("POST /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientGrantConsentHandler), "/api/v1/patient/consents"))
	mux.Handle("DELETE /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientRevokeConsentHandler), "/api/v1/patient/consents"))
	mux.Handle("POST /api/v1/patient/records/answer", tracing.WrapHandlerFunc(corsMiddleware(PatientHealthAnswerHandler), "/api/v1/patient/records/answer"))
	mux.Handle("GET /api/v1/patient/calls", tracing.WrapHandlerFunc(corsMiddleware(PatientCallsHandler), "/api/v1/patient/calls"))
	mux.Handle("GET /api/v1/patient/audit", tracing.WrapHandlerFunc(corsMiddleware(PatientAuditHandler), "/api/v1/patient/audit"))
	mux.Handle("GET /api/v1/hospital/incidents", tracing.WrapHandlerFunc(corsMiddleware(HospitalIncidentsHandler), "/api/v1/hospital/incidents"))
	mux.Handle("GET /api/v1/hospital/patient/audit", tracing.WrapHandlerFunc(corsMiddleware(HospitalPatientAuditHandler), "/api/v1/hospital/patient/audit"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: otelhttp.NewHandler(mux, "api-gateway"),
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
