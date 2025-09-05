#!/bin/sh

# Generate nginx configuration dynamically based on environment variables
# This script extracts the store API hostname from REACT_APP_STORE_API_URL

# Extract hostname from REACT_APP_STORE_API_URL (e.g., http://store-s1:8083 -> store-s1:8083)
STORE_API_HOST=$(echo "$REACT_APP_STORE_API_URL" | sed 's|http://||' | sed 's|https://||')

echo "Generating nginx config for store API: $STORE_API_HOST"

# Generate nginx configuration
cat > /etc/nginx/conf.d/default.conf << EOF
server {
    listen 80;
    server_name localhost;
    
    # Serve static files
    location / {
        root /usr/share/nginx/html;
        index index.html;
        try_files \$uri \$uri/ /index.html;
        
        # Cache static assets
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        add_header Pragma "no-cache";
        add_header Expires "0";
    }

    # Proxy API requests to the correct Store API
    # Frontend calls /api/inventory/* -> Store API /v1/store/inventory/*
    location /api/ {
        # Remove /api prefix and proxy to store API with /v1/store prefix
        rewrite ^/api/(.*)$ /v1/store/\$1 break;
        proxy_pass http://$STORE_API_HOST;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        
        # Add CORS headers
        add_header Access-Control-Allow-Origin *;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS";
        add_header Access-Control-Allow-Headers "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,X-API-Key";
        
        # Handle preflight requests
        if (\$request_method = 'OPTIONS') {
            add_header Access-Control-Allow-Origin *;
            add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS";
            add_header Access-Control-Allow-Headers "DNT,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Range,X-API-Key";
            add_header Access-Control-Max-Age 1728000;
            add_header Content-Type 'text/plain; charset=utf-8';
            add_header Content-Length 0;
            return 204;
        }
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF

echo "Nginx configuration generated successfully!"
echo "Store API configured to proxy to: $STORE_API_HOST"
