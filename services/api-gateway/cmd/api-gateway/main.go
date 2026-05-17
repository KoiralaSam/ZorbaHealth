package main

import (
	"context"
	"log"
	"net/http"

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

	mux := http.NewServeMux()

	// CORS preflight: OPTIONS must be handled for each path (browser sends OPTIONS before POST)
	optCORS := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify-otp", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/login", optCORS)

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.Handle("POST /api/v1/auth/patient/login", tracing.WrapHandlerFunc(corsMiddleware(PatientLoginHandler), "/api/v1/auth/patient/login"))
	mux.Handle("POST /api/v1/auth/patient/register", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterHandler), "/api/v1/auth/patient/register"))
	mux.Handle("POST /api/v1/auth/patient/register/verify", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyHandler), "/api/v1/auth/patient/register/verify"))
	mux.Handle("POST /api/v1/auth/patient/register/verify-otp", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyOTPHandler), "/api/v1/auth/patient/register/verify-otp"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: otelhttp.NewHandler(mux, "api-gateway"),
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
