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

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(logger, inventoryClient, serviceName, version)
	inventoryHandler := handlers.NewInventoryHandler(logger, inventoryClient)

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

		// Store-specific inventory endpoints
		r.Get("/store/inventory", inventoryHandler.GetAllProducts)
		r.Get("/store/inventory/{productId}", inventoryHandler.GetProduct)
		r.Post("/store/inventory/updates", inventoryHandler.UpdateInventory)
		r.Post("/store/inventory/batch-updates", inventoryHandler.BatchUpdateInventory)
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

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
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
