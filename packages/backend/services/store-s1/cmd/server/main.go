package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/melibackend/shared/client"
	sharedmiddleware "github.com/melibackend/shared/middleware"
	"github.com/melibackend/shared/storage"
	"github.com/melibackend/shared/sync"
	"github.com/melibackend/shared/utils"
	"github.com/melibackend/store-s1/internal/config"
	"github.com/melibackend/store-s1/internal/handlers"
)

const (
	serviceName = "store-s1-api"
	version     = "1.0.0"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize logger
	logger := utils.NewLogger(utils.LogLevel(cfg.LogLevel))

	logger.Info("Starting Store S1 API",
		"service", serviceName,
		"version", version,
		"port", cfg.Port,
		"environment", cfg.Environment,
		"central_api_url", cfg.CentralAPIURL,
	)

	// Initialize inventory client
	inventoryClient := client.NewInventoryClient(cfg.CentralAPIURL, cfg.CentralAPIKey)

	// Test connection to central API
	if _, err := inventoryClient.HealthCheck(); err != nil {
		logger.Error("Failed to connect to central inventory API", "error", err)
		os.Exit(1)
	}
	logger.Info("Successfully connected to central inventory API")

	// Initialize local storage
	localStorage := storage.NewMemoryStorage(cfg.DataDir)
	if err := localStorage.Initialize(); err != nil {
		logger.Error("Failed to initialize local storage", "error", err)
		os.Exit(1)
	}
	logger.Info("Local storage initialized", "data_dir", cfg.DataDir)

	// Initialize sync manager
	syncManager := sync.NewManager(inventoryClient, localStorage, logger)
	syncManager.SetSyncInterval(time.Duration(cfg.SyncInterval) * time.Minute)

	// Start sync manager with initial sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := syncManager.Start(ctx); err != nil {
		logger.Error("Failed to start sync manager", "error", err)
		os.Exit(1)
	}
	logger.Info("Sync manager started successfully")

	// Initialize handlers with local storage
	healthHandler := handlers.NewHealthHandler(logger, inventoryClient, serviceName, version)
	inventoryHandler := handlers.NewInventoryHandler(logger, inventoryClient, localStorage, syncManager)

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
	authMiddleware := sharedmiddleware.AuthMiddleware(apiKeys, logger)

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

		logger.Info("Shutting down server...")

		// Stop sync manager
		syncManager.Stop()

		// Close local storage
		if err := localStorage.Close(); err != nil {
			logger.Error("Failed to close local storage", "error", err)
		}

		// Cancel context
		cancel()

		// Shutdown server
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", "error", err)
		}
	}()

	logger.Info("Server ready to accept connections", "address", server.Addr)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Server failed to start", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped")
}
