package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"forecaster/internal/api"
	"forecaster/internal/config"
	"forecaster/internal/forecast"
)

func main() {
	// Resolve sites path: SITES_PATH env var wins over -sites flag default.
	sitesPath := flag.String("sites", "sites.json", "path to sites JSON file")
	flag.Parse()
	if v := os.Getenv("SITES_PATH"); v != "" {
		*sitesPath = v
	}

	sites, err := config.LoadSites(*sitesPath)
	if err != nil {
		log.Fatalf("loading sites: %v", err)
	}

	cache := forecast.NewCache()
	handler := api.NewHandler(sites, cache)

	mux := http.NewServeMux()

	// API routes first — must match before static file server
	mux.HandleFunc("GET /api/forecast", handler.ServeHTTP)
	mux.HandleFunc("GET /healthz", handler.Healthz)

	// Serve frontend assets
	mux.Handle("/", http.FileServer(http.Dir("public")))

	addr := ":8080"
	log.Printf("Starting forecaster server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
