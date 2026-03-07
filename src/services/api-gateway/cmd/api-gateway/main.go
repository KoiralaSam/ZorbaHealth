package main

import (
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

var (
	httpAddr = env.GetString("API_GATEWAY_HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")

	mux := http.NewServeMux()

	// CORS preflight: OPTIONS must be handled for each path (browser sends OPTIONS before POST)
	optCORS := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/login", optCORS)

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.HandleFunc("POST /api/v1/auth/patient/login", corsMiddleware(PatientLoginHandler))
	mux.HandleFunc("POST /api/v1/auth/patient/register", corsMiddleware(PatientRegisterHandler))
	mux.HandleFunc("POST /api/v1/auth/patient/register/verify", corsMiddleware(PatientRegisterVerifyHandler))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
