 #!/bin/sh

echo "🚀 Starting Store Frontend"
echo "=========================="
echo "Store ID: $REACT_APP_STORE_ID"
echo "Store Name: $REACT_APP_STORE_NAME"
echo "Store API URL: $REACT_APP_STORE_API_URL"
echo "API Base URL: $REACT_APP_API_BASE_URL"
echo ""

# Generate nginx configuration based on environment variables
echo "📝 Generating nginx configuration..."
/generate-nginx-config.sh

# Validate nginx configuration
echo "✅ Validating nginx configuration..."
nginx -t

if [ $? -eq 0 ]; then
    echo "✅ Nginx configuration is valid"
    echo ""
    echo "🌐 Starting nginx server..."
    echo "Frontend will be available on port 80"
    echo "API requests to /api/* will be proxied to: $REACT_APP_STORE_API_URL/v1/store/*"
    echo ""
    
    # Start nginx in foreground
    exec nginx -g "daemon off;"
else
    echo "❌ Nginx configuration is invalid!"
    exit 1
fi
