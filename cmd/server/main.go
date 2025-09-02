package main

import (
	"log/slog"
	"net/http"

	"inventory-management-api/internal/config"
	"inventory-management-api/internal/handlers"
	"inventory-management-api/internal/middleware"
	"inventory-management-api/internal/services"

	"github.com/gorilla/mux"
)

func main() {
	// Load configuration from .env file and environment variables
	cfg := config.LoadConfig()

	slog.Info("Starting Inventory Management API", "version", "1.0.0")

	r := mux.NewRouter()

	// Initialize services
	inventoryService, err := services.NewInventoryService()
	if err != nil {
		slog.Error("Failed to initialize inventory service", "error", err)
		return
	}
	slog.Info("Inventory service initialized successfully")

	// Initialize handlers
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)
	healthHandler := handlers.NewHealthHandler()
	slog.Debug("HTTP handlers initialized")

	// Apply auth middleware to v1 API routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.AuthMiddleware)

	// Central Inventory API routes (v1)
	v1.HandleFunc("/inventory/updates", inventoryHandler.UpdateInventory).Methods("POST")
	v1.HandleFunc("/inventory/{productId}", inventoryHandler.GetProduct).Methods("GET")
	v1.HandleFunc("/inventory", inventoryHandler.ListProducts).Methods("GET")

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", healthHandler.Health).Methods("GET")

	slog.Info("Starting HTTP server",
		"port", cfg.Port,
		"environment", cfg.Environment)

	slog.Debug("Available endpoints",
		"v1_endpoints", []string{
			"POST /v1/inventory/updates (single & batch)",
			"GET /v1/inventory/{productId}",
			"GET /v1/inventory (with replication support)",
		},
		"replication_params", []string{
			"?snapshot=true (full state)",
			"?since=<offset> (changes)",
			"?format=replication (metadata)",
		},
		"system_endpoints", []string{
			"GET /health",
		})

	slog.Info("Server ready to accept connections", "address", ":"+cfg.Port)

	err = http.ListenAndServe(":"+cfg.Port, r)
	if err != nil {
		slog.Error("Server failed to start", "error", err)
	}
}
