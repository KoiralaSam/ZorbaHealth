package main

import (
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
)

var (
	httpAddr = env.GetString("HEALTH_PROVIDER_SERVICE_HTTP_ADDR", ":8084")
)

func main() {
	mux := http.NewServeMux()

	// TODO: wire health-provider HTTP handlers here (provider registration, org profile, etc.).

	log.Printf("Health Provider Service listening on %s", httpAddr)

	server := &http.Server{
		Addr:    httpAddr,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}

