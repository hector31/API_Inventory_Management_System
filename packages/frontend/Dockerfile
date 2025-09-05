# 🚀 Ultra-Optimized Dockerfile for Static Frontend
# ✅ No npm install needed - Build in seconds, not minutes!
# ✅ Full environment variable support
# ✅ Multi-stage build for minimal image size

# Stage 1: Process static files with environment variables
FROM node:18-alpine AS builder

WORKDIR /app

# 📋 Build arguments for environment variables
ARG REACT_APP_API_BASE_URL=/api
ARG REACT_APP_STORE_API_URL=http://store-s1:8083
ARG REACT_APP_API_KEY=demo
ARG REACT_APP_STORE_ID=store-s1
ARG REACT_APP_STORE_NAME=Store S1
ARG REACT_APP_AUTO_REFRESH_INTERVAL=30000
ARG REACT_APP_REQUEST_TIMEOUT=10000

# 🔧 Set environment variables for build script
ENV REACT_APP_API_BASE_URL=$REACT_APP_API_BASE_URL
ENV REACT_APP_STORE_API_URL=$REACT_APP_STORE_API_URL
ENV REACT_APP_API_KEY=$REACT_APP_API_KEY
ENV REACT_APP_STORE_ID=$REACT_APP_STORE_ID
ENV REACT_APP_STORE_NAME=$REACT_APP_STORE_NAME
ENV REACT_APP_AUTO_REFRESH_INTERVAL=$REACT_APP_AUTO_REFRESH_INTERVAL
ENV REACT_APP_REQUEST_TIMEOUT=$REACT_APP_REQUEST_TIMEOUT

# 📁 Copy only essential files (NO package.json = NO npm install!)
COPY build-static.js ./
COPY public/ ./public/

# ⚡ Process static files with environment variables (ultra-fast!)
RUN node build-static.js

# Stage 2: Production nginx server
FROM nginx:alpine

# 📦 Copy processed static files
COPY --from=builder /app/build /usr/share/nginx/html

# 🔧 Copy nginx configuration scripts
COPY generate-nginx-config.sh /generate-nginx-config.sh
COPY start-nginx.sh /start-nginx.sh

# Make scripts executable
RUN chmod +x /generate-nginx-config.sh /start-nginx.sh

# 🏥 Install curl for health checks
RUN apk add --no-cache curl

# 📋 Build arguments for runtime environment variables
ARG REACT_APP_STORE_API_URL=http://store-s1:8083
ARG REACT_APP_STORE_ID=store-s1
ARG REACT_APP_STORE_NAME=Store S1
ARG REACT_APP_API_BASE_URL=/api

# 🔧 Set environment variables for runtime
ENV REACT_APP_STORE_API_URL=$REACT_APP_STORE_API_URL
ENV REACT_APP_STORE_ID=$REACT_APP_STORE_ID
ENV REACT_APP_STORE_NAME=$REACT_APP_STORE_NAME
ENV REACT_APP_API_BASE_URL=$REACT_APP_API_BASE_URL

# 🌐 Expose port
EXPOSE 80

# ❤️ Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# 🚀 Start nginx with dynamic configuration
CMD ["/start-nginx.sh"]
