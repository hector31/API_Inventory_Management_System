package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"inventory-management-api/internal/config"
	"inventory-management-api/internal/events"
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
	inventoryService, err := services.NewInventoryService(cfg)
	if err != nil {
		slog.Error("Failed to initialize inventory service", "error", err)
		return
	}
	slog.Info("Inventory service initialized successfully")

	// Initialize event queue
	maxEvents, _ := strconv.Atoi(cfg.MaxEventsInQueue)
	if maxEvents <= 0 {
		maxEvents = 10000
	}

	eventQueue, err := events.NewEventQueue(events.EventQueueConfig{
		FilePath:  cfg.EventsFilePath,
		MaxEvents: maxEvents,
		Logger:    slog.Default(),
	})
	if err != nil {
		slog.Error("Failed to initialize event queue", "error", err)
		return
	}
	slog.Info("Event queue initialized successfully")

	// Set event queue in inventory service for event publishing
	inventoryService.SetEventQueue(eventQueue)

	// Initialize handlers
	inventoryHandler := handlers.NewInventoryHandler(inventoryService)
	eventsHandler := handlers.NewEventsHandler(eventQueue, slog.Default())
	healthHandler := handlers.NewHealthHandler()
	slog.Debug("HTTP handlers initialized")

	// Apply auth middleware to v1 API routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.AuthMiddleware)

	// Central Inventory API routes (v1) - specific routes first
	v1.HandleFunc("/inventory/updates", inventoryHandler.UpdateInventory).Methods("POST")
	v1.HandleFunc("/inventory/events", eventsHandler.GetEvents).Methods("GET")
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
			"GET /v1/inventory/events (event streaming)",
		},
		"replication_params", []string{
			"?snapshot=true (full state)",
			"?since=<offset> (changes)",
			"?format=replication (metadata)",
		},
		"events_params", []string{
			"?offset=<number> (required: starting offset)",
			"?limit=<number> (optional: max events, default 100)",
			"?wait=<seconds> (optional: long polling, default 0)",
		},
		"system_endpoints", []string{
			"GET /health",
		})

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in a goroutine
	go func() {
		slog.Info("Server ready to accept connections", "address", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown event queue first
	if err := eventQueue.Close(); err != nil {
		slog.Error("Error closing event queue", "error", err)
	}

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("Server forced to shutdown", "error", err)
	}

	slog.Info("Server exited")
}
