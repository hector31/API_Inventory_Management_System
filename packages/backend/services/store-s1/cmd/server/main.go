package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/melibackend/shared/client"
	sharedmiddleware "github.com/melibackend/shared/middleware"
	"github.com/melibackend/shared/storage"
	"github.com/melibackend/shared/sync"
	"github.com/melibackend/store-s1/internal/config"
	"github.com/melibackend/store-s1/internal/handlers"
)

const (
	serviceName = "store-s1-api"
	version     = "1.0.0"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, so we just log if it's not found
		fmt.Printf("No .env file found or error loading it: %v\n", err)
	}

	// Load configuration (setupLogging is called automatically inside)
	cfg := config.Load()

	slog.Info("Starting Store S1 API",
		"service", serviceName,
		"version", version,
		"port", cfg.Port,
		"environment", cfg.Environment,
		"central_api_url", cfg.CentralAPIURL,
		"sync_interval_seconds", cfg.SyncIntervalSeconds,
		"event_wait_timeout_seconds", cfg.EventWaitTimeoutSeconds,
		"event_batch_limit", cfg.EventBatchLimit,
	)

	// Initialize inventory client
	inventoryClient := client.NewInventoryClient(cfg.CentralAPIURL, cfg.CentralAPIKey)

	// Test connection to central API
	if _, err := inventoryClient.HealthCheck(); err != nil {
		slog.Error("Failed to connect to central inventory API", "error", err)
		os.Exit(1)
	}
	slog.Info("Successfully connected to central inventory API")

	// Initialize local storage
	localStorage := storage.NewMemoryStorage(cfg.DataDir)
	if err := localStorage.Initialize(); err != nil {
		slog.Error("Failed to initialize local storage", "error", err)
		os.Exit(1)
	}
	slog.Info("Local storage initialized", "data_dir", cfg.DataDir)

	// Initialize event-driven sync manager
	eventSyncConfig := sync.EventSyncConfig{
		SyncIntervalSeconds:     cfg.SyncIntervalSeconds,
		EventWaitTimeoutSeconds: cfg.EventWaitTimeoutSeconds,
		EventBatchLimit:         cfg.EventBatchLimit,
		MaxConsecutiveFailures:  5, // Allow 5 consecutive failures before fallback
	}
	syncManager := sync.NewEventSyncManager(inventoryClient, localStorage, eventSyncConfig)

	// Start sync manager with initial sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := syncManager.Start(ctx); err != nil {
		slog.Error("Failed to start sync manager", "error", err)
		os.Exit(1)
	}
	slog.Info("Sync manager started successfully")

	// Initialize handlers with local storage
	healthHandler := handlers.NewHealthHandler(inventoryClient, serviceName, version)
	inventoryHandler := handlers.NewInventoryHandler(inventoryClient, localStorage, syncManager)

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// API Key authentication for protected routes
	apiKeys := strings.Split(cfg.APIKeys, ",")
	authMiddleware := sharedmiddleware.AuthMiddleware(apiKeys)

	// Routes
	r.Get("/health", healthHandler.HealthCheck)

	// Protected routes
	r.Route("/v1", func(r chi.Router) {
		r.Use(authMiddleware)

		// Store-specific inventory endpoints (now using local cache)
		r.Get("/store/inventory", inventoryHandler.GetAllProducts)
		r.Get("/store/inventory/{productId}", inventoryHandler.GetProduct)
		r.Post("/store/inventory/updates", inventoryHandler.UpdateInventory)
		r.Post("/store/inventory/batch-updates", inventoryHandler.BatchUpdateInventory)

		// Sync management endpoints
		r.Get("/store/sync/status", inventoryHandler.GetSyncStatus)
		r.Post("/store/sync/force", inventoryHandler.ForceSync)
		r.Get("/store/cache/stats", inventoryHandler.GetCacheStats)
	})

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		slog.Info("Shutting down server...")

		// Stop sync manager
		syncManager.Stop()

		// Close local storage
		if err := localStorage.Close(); err != nil {
			slog.Error("Failed to close local storage", "error", err)
		}

		// Cancel context
		cancel()

		// Shutdown server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
	}()

	slog.Info("Server ready to accept connections", "address", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}

	slog.Info("Server stopped")
}
