#!/bin/bash

# 🏪 Script para agregar una nueva tienda (backend + frontend)
# Uso: ./add-new-store.sh <store-number> [store-name]

set -e

# Validar argumentos
if [ $# -lt 1 ]; then
    echo "❌ Error: Debes proporcionar el número de la tienda"
    echo "📋 Uso: $0 <store-number> [store-name]"
    echo "📋 Ejemplo: $0 6"
    echo "📋 Ejemplo: $0 7 'Store Premium'"
    exit 1
fi

STORE_NUMBER=$1
STORE_NAME=${2:-"Store S${STORE_NUMBER}"}
STORE_ID="store-s${STORE_NUMBER}"
API_KEY="store-s${STORE_NUMBER}-key"

# Calcular puertos (incrementales)
BACKEND_PORT=$((8080 + STORE_NUMBER + 2))  # 8083, 8087, 8088, etc.
FRONTEND_PORT=$((3010 + STORE_NUMBER - 1)) # 3010, 3014, 3015, etc.

echo "🏪 Agregando nueva tienda:"
echo "========================="
echo "📋 Store Number: S${STORE_NUMBER}"
echo "🏷️  Store Name: ${STORE_NAME}"
echo "🆔 Store ID: ${STORE_ID}"
echo "🔑 API Key: ${API_KEY}"
echo "🌐 Backend Port: ${BACKEND_PORT}"
echo "🖥️  Frontend Port: ${FRONTEND_PORT}"
echo ""

# Verificar que los puertos no estén en uso
if netstat -tuln 2>/dev/null | grep -q ":${BACKEND_PORT} "; then
    echo "❌ Error: Puerto ${BACKEND_PORT} ya está en uso"
    exit 1
fi

if netstat -tuln 2>/dev/null | grep -q ":${FRONTEND_PORT} "; then
    echo "❌ Error: Puerto ${FRONTEND_PORT} ya está en uso"
    exit 1
fi

# Crear backup del docker-compose.yml
echo "💾 Creando backup de docker-compose.yml..."
cp docker-compose.yml docker-compose.yml.backup

# Generar configuración del backend
BACKEND_CONFIG="
  # 🆕 Store S${STORE_NUMBER} API
  store-s${STORE_NUMBER}:
    build:
      context: ./packages/backend
      dockerfile: ./services/store-s1/Dockerfile  # Reutiliza el mismo Dockerfile
    container_name: store-s${STORE_NUMBER}
    restart: unless-stopped
    ports:
      - \"${BACKEND_PORT}:8083\"
    environment:
      - PORT=8083
      - LOG_LEVEL=debug
      - ENVIRONMENT=development
      - API_KEYS=${API_KEY},demo
      - CENTRAL_API_URL=http://inventory-management-system:8081
      - CENTRAL_API_KEY=demo
      - DATA_DIR=/app/data
      # Legacy full sync interval (fallback)
      - SYNC_INTERVAL_MINUTES=5
      # Event-driven sync configuration
      - SYNC_INTERVAL_SECONDS=30
      - EVENT_WAIT_TIMEOUT_SECONDS=20
      - EVENT_BATCH_LIMIT=100
    networks:
      - meli_network
    depends_on:
      - inventory-management-system
    healthcheck:
      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost:8083/health\"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
"

# Generar configuración del frontend
FRONTEND_CONFIG="
  # 🆕 Store S${STORE_NUMBER} Frontend
  frontend-s${STORE_NUMBER}:
    build:
      context: ./packages/frontend
      dockerfile: Dockerfile
      args:
        - REACT_APP_API_BASE_URL=/api
        - REACT_APP_STORE_API_URL=http://store-s${STORE_NUMBER}:8083
        - REACT_APP_API_KEY=demo
        - REACT_APP_STORE_ID=${STORE_ID}
        - REACT_APP_STORE_NAME=${STORE_NAME}
        - REACT_APP_AUTO_REFRESH_INTERVAL=30000
        - REACT_APP_REQUEST_TIMEOUT=10000
    container_name: store-s${STORE_NUMBER}-frontend
    restart: unless-stopped
    ports:
      - \"${FRONTEND_PORT}:80\"
    depends_on:
      - store-s${STORE_NUMBER}
    networks:
      - meli_network
    healthcheck:
      test: [\"CMD\", \"curl\", \"-f\", \"http://localhost/\"]
      interval: 30s
      timeout: 10s
      retries: 3
"

# Encontrar la línea donde insertar el backend (antes de "# Frontend Application")
BACKEND_INSERT_LINE=$(grep -n "# Frontend Application" docker-compose.yml | head -1 | cut -d: -f1)
BACKEND_INSERT_LINE=$((BACKEND_INSERT_LINE - 1))

# Encontrar la línea donde insertar el frontend (antes de "# OpenTelemetry Collector")
FRONTEND_INSERT_LINE=$(grep -n "# OpenTelemetry Collector" docker-compose.yml | head -1 | cut -d: -f1)
FRONTEND_INSERT_LINE=$((FRONTEND_INSERT_LINE - 1))

echo "🔧 Agregando configuración del backend..."
# Insertar backend
head -n $BACKEND_INSERT_LINE docker-compose.yml > temp_compose.yml
echo "$BACKEND_CONFIG" >> temp_compose.yml
tail -n +$((BACKEND_INSERT_LINE + 1)) docker-compose.yml >> temp_compose.yml
mv temp_compose.yml docker-compose.yml

echo "🔧 Agregando configuración del frontend..."
# Recalcular línea del frontend (el archivo cambió)
FRONTEND_INSERT_LINE=$(grep -n "# OpenTelemetry Collector" docker-compose.yml | head -1 | cut -d: -f1)
FRONTEND_INSERT_LINE=$((FRONTEND_INSERT_LINE - 1))

# Insertar frontend
head -n $FRONTEND_INSERT_LINE docker-compose.yml > temp_compose.yml
echo "$FRONTEND_CONFIG" >> temp_compose.yml
tail -n +$((FRONTEND_INSERT_LINE + 1)) docker-compose.yml >> temp_compose.yml
mv temp_compose.yml docker-compose.yml

echo ""
echo "✅ Nueva tienda agregada exitosamente!"
echo ""
echo "🚀 Para construir y ejecutar:"
echo "============================="
echo "# Construir solo el backend:"
echo "docker-compose build store-s${STORE_NUMBER}"
echo ""
echo "# Construir solo el frontend:"
echo "docker-compose build frontend-s${STORE_NUMBER}"
echo ""
echo "# Construir ambos:"
echo "docker-compose build store-s${STORE_NUMBER} frontend-s${STORE_NUMBER}"
echo ""
echo "# Ejecutar la nueva tienda:"
echo "docker-compose up -d store-s${STORE_NUMBER} frontend-s${STORE_NUMBER}"
echo ""
echo "🌐 URLs de acceso:"
echo "=================="
echo "🔗 Backend API: http://localhost:${BACKEND_PORT}"
echo "🔗 Frontend: http://localhost:${FRONTEND_PORT}"
echo "🔗 Health Check: http://localhost:${BACKEND_PORT}/health"
echo ""
echo "📋 Información de la tienda:"
echo "============================"
echo "Store ID: ${STORE_ID}"
echo "Store Name: ${STORE_NAME}"
echo "API Key: ${API_KEY}"
echo ""
echo "💾 Backup guardado en: docker-compose.yml.backup"
