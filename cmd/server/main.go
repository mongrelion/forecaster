package main

import (
	"log"
	"net/http"

	"forecaster/internal/api"
	"forecaster/internal/forecast"
)

func main() {
	cache := forecast.NewCache()
	handler := api.NewHandler(cache)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/forecast", handler.ServeHTTP)

	addr := ":8080"
	log.Printf("Starting forecaster server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}