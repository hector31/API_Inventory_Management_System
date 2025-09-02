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

	// Apply auth middleware to v1 API routes
	v1 := r.PathPrefix("/v1").Subrouter()
	v1.Use(middleware.AuthMiddleware)

	// Central Inventory API routes (v1)
	v1.HandleFunc("/inventory/updates", inventoryHandler.UpdateInventory).Methods("POST")
	v1.HandleFunc("/inventory/sync", inventoryHandler.SyncInventory).Methods("POST")
	v1.HandleFunc("/inventory/{productId}", inventoryHandler.GetProduct).Methods("GET")
	v1.HandleFunc("/inventory/global/{productId}", inventoryHandler.GetGlobalAvailability).Methods("GET")
	v1.HandleFunc("/inventory", inventoryHandler.ListProducts).Methods("GET")

	// Replication API routes (v1)
	v1.HandleFunc("/replication/snapshot", replicationHandler.GetSnapshot).Methods("GET")
	v1.HandleFunc("/replication/changes", replicationHandler.GetChanges).Methods("GET")

	// Health check endpoint (no auth required)
	r.HandleFunc("/health", healthHandler.Health).Methods("GET")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fmt.Printf("Starting Central Inventory Management API server on port %s\n", port)
	fmt.Println("Available endpoints:")
	fmt.Println("  API v1:")
	fmt.Println("    POST /v1/inventory/updates - Mutate stock")
	fmt.Println("    POST /v1/inventory/sync - Bulk sync")
	fmt.Println("    GET  /v1/inventory/{productId} - Read product")
	fmt.Println("    GET  /v1/inventory/global/{productId} - Global availability")
	fmt.Println("    GET  /v1/inventory - List products")
	fmt.Println("    GET  /v1/replication/snapshot - Replication snapshot")
	fmt.Println("    GET  /v1/replication/changes - Replication changes")
	fmt.Println("  System:")
	fmt.Println("    GET  /health - Health check (unversioned)")

	log.Fatal(http.ListenAndServe(":"+port, r))
}
