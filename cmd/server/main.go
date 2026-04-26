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

	// API routes first — must match before static file server
	mux.HandleFunc("GET /api/forecast", handler.ServeHTTP)

	// Serve frontend assets
	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := ":8080"
	log.Printf("Starting forecaster server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
