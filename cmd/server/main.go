package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"inventory-management-api/internal/handlers"
	"inventory-management-api/internal/middleware"

	"github.com/gorilla/mux"
)

func main() {
	r := mux.NewRouter()

	// Initialize handlers
	inventoryHandler := handlers.NewInventoryHandler()
	replicationHandler := handlers.NewReplicationHandler()
	healthHandler := handlers.NewHealthHandler()

	// Apply auth middleware to protected routes
	api := r.PathPrefix("/").Subrouter()
	api.Use(middleware.AuthMiddleware)

	// Central Inventory API routes
	api.HandleFunc("/inventory/updates", inventoryHandler.UpdateInventory).Methods("POST")
	api.HandleFunc("/inventory/sync", inventoryHandler.SyncInventory).Methods("POST")
	api.HandleFunc("/inventory/{productId}", inventoryHandler.GetProduct).Methods("GET")
	api.HandleFunc("/inventory/global/{productId}", inventoryHandler.GetGlobalAvailability).Methods("GET")
	api.HandleFunc("/inventory", inventoryHandler.ListProducts).Methods("GET")

	// Replication API routes
	api.HandleFunc("/replication/snapshot", replicationHandler.GetSnapshot).Methods("GET")
	api.HandleFunc("/replication/changes", replicationHandler.GetChanges).Methods("GET")

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", healthHandler.Health).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Central Inventory Management API server on port %s\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  POST /inventory/updates - Mutate stock")
	fmt.Println("  POST /inventory/sync - Bulk sync")
	fmt.Println("  GET  /inventory/{productId} - Read product")
	fmt.Println("  GET  /inventory/global/{productId} - Global availability")
	fmt.Println("  GET  /inventory - List products")
	fmt.Println("  GET  /replication/snapshot - Replication snapshot")
	fmt.Println("  GET  /replication/changes - Replication changes")
	fmt.Println("  GET  /health - Health check")

	log.Fatal(http.ListenAndServe(":"+port, r))
}
