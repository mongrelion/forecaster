package main

import (
	"log"
	"net"
	"net/http"

	"forecaster/internal/api"
	"forecaster/internal/config"
	"forecaster/internal/forecast"
)

func main() {
	cfg := config.LoadServerConfig()

	sites, err := config.LoadSites(cfg.SitesPath)
	if err != nil {
		log.Fatalf("loading sites: %v", err)
	}

	cache := forecast.NewCache()
	handler := api.NewHandler(sites, cache, cfg)

	mux := http.NewServeMux()

	// API routes first — must match before static file server
	mux.HandleFunc("GET /api/forecast", handler.ServeHTTP)
	mux.HandleFunc("GET /healthz", handler.Healthz)

	// Serve frontend assets
	mux.Handle("/", http.FileServer(http.Dir(cfg.PublicDir)))

	addr := net.JoinHostPort(cfg.Host, cfg.Port)
	log.Printf("Starting forecaster server on %s, serving public dir %s", addr, cfg.PublicDir)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
