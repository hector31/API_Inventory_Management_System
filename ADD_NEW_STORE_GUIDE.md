# 🏪 Guía para Agregar Nuevas Tiendas

## 🚀 **Proceso Automatizado (Recomendado)**

### **Método 1: Script Automático**

```bash
# Agregar Store S6
./add-new-store.sh 6

# Agregar Store S7 con nombre personalizado
./add-new-store.sh 7 "Store Premium"

# Construir y probar la nueva tienda
./test-new-store.sh 6
```

### **Método 2: Manual (Paso a Paso)**

## 📋 **Información Necesaria para Nueva Tienda**

Para agregar Store S6, necesitas:

| Campo | Valor | Descripción |
|-------|-------|-------------|
| **Store Number** | 6 | Número de la tienda |
| **Store ID** | store-s6 | Identificador único |
| **Store Name** | Store S6 | Nombre mostrado en frontend |
| **API Key** | store-s6-key | Clave de API |
| **Backend Port** | 8088 | Puerto del API backend |
| **Frontend Port** | 3015 | Puerto del frontend |

## 🔧 **Paso 1: Agregar Backend en docker-compose.yml**

```yaml
  # 🆕 Store S6 API
  store-s6:
    build:
      context: ./packages/backend
      dockerfile: ./services/store-s1/Dockerfile  # Reutiliza Dockerfile existente
    container_name: store-s6
    restart: unless-stopped
    ports:
      - "8088:8083"  # Puerto externo único
    environment:
      - PORT=8083
      - LOG_LEVEL=debug
      - ENVIRONMENT=development
      - API_KEYS=store-s6-key,demo
      - CENTRAL_API_URL=http://inventory-management-system:8081
      - CENTRAL_API_KEY=demo
      - DATA_DIR=/app/data
      - SYNC_INTERVAL_MINUTES=5
      - SYNC_INTERVAL_SECONDS=30
      - EVENT_WAIT_TIMEOUT_SECONDS=20
      - EVENT_BATCH_LIMIT=100
    networks:
      - meli_network
    depends_on:
      - inventory-management-system
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8083/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

## 🖥️ **Paso 2: Agregar Frontend en docker-compose.yml**

```yaml
  # 🆕 Store S6 Frontend
  frontend-s6:
    build:
      context: ./packages/frontend
      dockerfile: Dockerfile  # Usa el Dockerfile optimizado
      args:
        - REACT_APP_API_BASE_URL=/api
        - REACT_APP_STORE_API_URL=http://store-s6:8083
        - REACT_APP_API_KEY=demo
        - REACT_APP_STORE_ID=store-s6
        - REACT_APP_STORE_NAME=Store S6
        - REACT_APP_AUTO_REFRESH_INTERVAL=30000
        - REACT_APP_REQUEST_TIMEOUT=10000
    container_name: store-s6-frontend
    restart: unless-stopped
    ports:
      - "3015:80"  # Puerto externo único
    depends_on:
      - store-s6
    networks:
      - meli_network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## 🚀 **Paso 3: Construir y Ejecutar**

```bash
# Construir servicios
docker-compose build store-s6 frontend-s6

# Ejecutar servicios
docker-compose up -d store-s6 frontend-s6

# Verificar estado
docker-compose ps store-s6 frontend-s6
```

## 🧪 **Paso 4: Verificar Funcionamiento**

### **Backend Tests:**
```bash
# Health check
curl http://localhost:8088/health

# Inventory endpoint
curl http://localhost:8088/api/inventory -H "X-API-Key: demo"

# Specific product
curl http://localhost:8088/api/inventory/PROD-001 -H "X-API-Key: demo"
```

### **Frontend Tests:**
```bash
# Accessibility
curl http://localhost:3015

# Store name verification
curl http://localhost:3015 | grep "Store S6"

# Manual browser test
open http://localhost:3015
```

## 📊 **Convenciones de Puertos**

| Store | Backend Port | Frontend Port | Pattern |
|-------|--------------|---------------|---------|
| S1 | 8083 | 3010 | Base |
| S2 | 8084 | 3011 | +1 |
| S3 | 8085 | 3012 | +1 |
| S4 | 8086 | 3013 | +1 |
| S5 | 8087 | 3014 | +1 |
| **S6** | **8088** | **3015** | **+1** |
| S7 | 8089 | 3016 | +1 |

## 🔄 **Ventajas del Approach Actual**

### **✅ Reutilización de Código:**
- **Un solo Dockerfile** para todos los backends
- **Un solo Dockerfile** para todos los frontends
- **Configuración por variables de entorno**

### **✅ Escalabilidad:**
- **Agregar tienda = 30 líneas en docker-compose.yml**
- **No modificar código fuente**
- **Build ultra-rápido (segundos)**

### **✅ Mantenimiento:**
- **Actualizaciones centralizadas**
- **Configuración consistente**
- **Fácil debugging**

## 🎯 **Casos de Uso Avanzados**

### **Tienda con Configuración Especial:**
```yaml
  # Store Premium con configuración especial
  store-premium:
    build:
      context: ./packages/backend
      dockerfile: ./services/store-s1/Dockerfile
    environment:
      - API_KEYS=premium-key,demo
      - SYNC_INTERVAL_SECONDS=10  # Sync más frecuente
      - EVENT_BATCH_LIMIT=200     # Batches más grandes
    ports:
      - "8090:8083"
```

### **Frontend con Tema Personalizado:**
```yaml
  frontend-premium:
    build:
      context: ./packages/frontend
      dockerfile: Dockerfile
      args:
        - REACT_APP_STORE_NAME=Premium Store
        - REACT_APP_AUTO_REFRESH_INTERVAL=15000  # Refresh más frecuente
    ports:
      - "3020:80"
```

## 🛠️ **Comandos Útiles**

```bash
# Ver todas las tiendas
docker-compose ps | grep store

# Logs de una tienda específica
docker-compose logs -f store-s6

# Restart una tienda
docker-compose restart store-s6 frontend-s6

# Rebuild una tienda
docker-compose build --no-cache store-s6 frontend-s6

# Eliminar una tienda
docker-compose stop store-s6 frontend-s6
docker-compose rm store-s6 frontend-s6
```

## 🎉 **Resultado Final**

Con este approach puedes:
- ✅ **Agregar tiendas en minutos**
- ✅ **Mantener código centralizado**
- ✅ **Escalar horizontalmente**
- ✅ **Configurar individualmente**
- ✅ **Build ultra-rápido**

¡Cada nueva tienda es completamente funcional con su propio backend, frontend y configuración única!
