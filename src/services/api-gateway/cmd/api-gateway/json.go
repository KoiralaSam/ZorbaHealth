package main

import (
	"encoding/json"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
)

// Helper function to send responses
func writeJson(w http.ResponseWriter, statusCode int, Data any, error *contracts.APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := contracts.APIResponse{
		Data:  Data,
		Error: error,
	}

	json.NewEncoder(w).Encode(response)
}
