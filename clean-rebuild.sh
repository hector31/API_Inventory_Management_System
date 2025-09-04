#!/bin/bash

echo "🧹 Cleaning Docker cache and rebuilding..."

# Stop and remove containers
echo "Stopping containers..."
docker-compose down

# Remove images
echo "Removing old images..."
docker-compose down --rmi all

# Clean Docker cache
echo "Cleaning Docker cache..."
docker system prune -f

# Rebuild with no cache
echo "Rebuilding inventory-management-system..."
docker-compose build --no-cache inventory-management-system

echo "Rebuilding store-s1..."
docker-compose build --no-cache store-s1

echo "✅ Clean rebuild complete!"
