# 🏪 Store S1 Frontend Application

A React-based frontend application for the Store S1 inventory management system. This application provides a user-friendly interface for browsing products, managing shopping cart, and making purchases through the Store S1 API.

## 🚀 Features

### Core Functionality
- **Product Listing**: Browse all available products from the store's local cache
- **Real-time Inventory**: Live updates of product availability and quantities
- **Shopping Cart**: Add, remove, and modify quantities of products
- **Purchase Processing**: Complete purchases through the Store S1 API
- **Responsive Design**: Works on desktop, tablet, and mobile devices

### Advanced Features
- **Search & Filter**: Find products by ID or name
- **Sorting**: Sort products by ID, stock level, or other criteria
- **Pagination**: Navigate through large product catalogs efficiently
- **Stock Warnings**: Visual indicators for low stock and out-of-stock items
- **Sync Status**: Real-time display of synchronization status with central API
- **Auto-refresh**: Automatic updates every 30 seconds
- **Error Handling**: Comprehensive error messages and retry mechanisms

## 🏗️ Architecture

### Technology Stack
- **Frontend Framework**: React 18
- **Styling**: Custom CSS with responsive design
- **HTTP Client**: Fetch API with custom error handling
- **Build Tool**: Create React App
- **Production Server**: Nginx with API proxy

### API Integration
- **Primary API**: Store S1 API (port 8083)
- **Authentication**: X-API-Key header authentication
- **Endpoints Used**:
  - `GET /v1/store/inventory` - Product listing
  - `PUT /v1/store/inventory` - Purchase processing
  - `GET /v1/store/sync/status` - Sync status
  - `POST /v1/store/sync/force` - Force synchronization

### Docker Integration
- **Multi-stage build**: Optimized for production
- **Nginx proxy**: Routes API calls to Store S1 backend
- **Health checks**: Monitoring and auto-recovery
- **Network isolation**: Secure container communication

## 🚀 Quick Start

### Using Docker (Recommended)
```bash
# Start all services including frontend
docker-compose up -d

# Access the application
open http://localhost:3002
```

### Development Mode
```bash
cd packages/frontend

# Install dependencies
npm install

# Start development server
npm start

# Access at http://localhost:3000
```

## 📱 User Interface

### Header Section
- **Store branding** and title
- **Sync status indicator** with real-time updates
- **Cart summary** showing total items
- **Action buttons** for refresh and force sync
- **Last update timestamp**

### Product Grid
- **Card-based layout** with product information
- **Stock status badges** (In Stock, Low Stock, Out of Stock)
- **Quantity selectors** with validation
- **Add to cart functionality** with real-time feedback
- **Search and sort controls**

### Shopping Cart
- **Sticky sidebar** for easy access
- **Item management** with quantity controls
- **Stock validation** and warnings
- **Total calculations** with pricing
- **Purchase processing** with loading states

### Responsive Design
- **Desktop**: Full grid layout with sidebar cart
- **Tablet**: Stacked layout with optimized spacing
- **Mobile**: Single column with touch-friendly controls

## 🔧 Configuration

### Environment Variables
```bash
NODE_ENV=production          # Production mode
```

### API Configuration
```javascript
// Development
API_BASE_URL=http://localhost:8083/v1/store

// Production (Docker)
API_BASE_URL=/api  # Proxied through nginx
```

### Nginx Proxy Configuration
```nginx
# API proxy to Store S1
location /api/ {
    proxy_pass http://store-s1:8083/v1/store/;
    # CORS and headers configuration
}
```

## 🧪 Testing the Application

### Manual Testing Workflow
1. **Access the application**: http://localhost:3002
2. **Verify product loading**: Products should appear in the grid
3. **Test search functionality**: Search for specific product IDs
4. **Add items to cart**: Use quantity selectors and add to cart
5. **Modify cart contents**: Adjust quantities in the cart sidebar
6. **Process purchase**: Click purchase button and verify success
7. **Check real-time updates**: Verify inventory updates after purchase

### API Testing
```bash
# Test product listing
curl http://localhost:3002/api/inventory

# Test health check
curl http://localhost:3002/health
```

### Performance Testing
- **Load time**: Initial page load should be < 2 seconds
- **API response**: Product listing should load < 1 second
- **Real-time updates**: Auto-refresh every 30 seconds
- **Purchase processing**: Should complete < 3 seconds

## 🐛 Troubleshooting

### Common Issues

#### Frontend not loading
```bash
# Check container status
docker-compose ps frontend

# Check logs
docker-compose logs frontend

# Verify port mapping
curl http://localhost:3002/health
```

#### API connection issues
```bash
# Test Store S1 API directly
curl http://localhost:8083/health

# Check nginx proxy configuration
docker-compose exec frontend cat /etc/nginx/conf.d/default.conf
```

#### Products not loading
```bash
# Check API key configuration
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/inventory

# Verify Store S1 sync status
curl -H "X-API-Key: demo" http://localhost:8083/v1/store/sync/status
```

### Debug Mode
```bash
# Enable debug logging in browser console
localStorage.setItem('debug', 'true');

# View network requests in browser DevTools
# Check Console tab for error messages
```

## 📊 Monitoring

### Health Checks
- **Frontend health**: http://localhost:3002/health
- **API connectivity**: Automatic validation on page load
- **Sync status**: Real-time monitoring in header

### Performance Metrics
- **Page load time**: Measured in browser DevTools
- **API response time**: Displayed in network tab
- **Error rates**: Logged to browser console
- **User interactions**: Cart operations and purchases

### Logs
```bash
# Frontend container logs
docker-compose logs -f frontend

# Nginx access logs
docker-compose exec frontend tail -f /var/log/nginx/access.log

# Application errors in browser console
```

## 🔐 Security

### Authentication
- **API Key**: Required for all Store S1 API calls
- **CORS**: Configured for secure cross-origin requests
- **Input validation**: Client-side validation for all user inputs

### Data Protection
- **No sensitive data storage**: No local storage of sensitive information
- **Secure communication**: All API calls over HTTP (HTTPS in production)
- **Error handling**: No sensitive information in error messages

## 🚀 Production Deployment

### Docker Production Build
```bash
# Build production image
docker-compose build frontend

# Deploy with production settings
docker-compose up -d frontend
```

### Performance Optimizations
- **Static asset caching**: Nginx serves static files efficiently
- **Gzip compression**: Enabled for all text assets
- **Bundle optimization**: React production build with minification
- **Image optimization**: Multi-stage Docker build for smaller images

### Scaling Considerations
- **Horizontal scaling**: Multiple frontend containers behind load balancer
- **CDN integration**: Static assets served from CDN
- **API rate limiting**: Respect Store S1 API limits
- **Caching strategy**: Browser caching for static assets

## 📈 Future Enhancements

### Planned Features
- **WebSocket integration**: Real-time inventory updates
- **User authentication**: Individual user accounts and preferences
- **Order history**: Track previous purchases
- **Product images**: Visual product catalog
- **Advanced filtering**: Category, price range, availability filters
- **Bulk operations**: Multi-product purchase workflows

### Technical Improvements
- **TypeScript migration**: Type safety and better development experience
- **State management**: Redux or Context API for complex state
- **Testing suite**: Unit tests, integration tests, E2E tests
- **PWA features**: Offline support and mobile app-like experience
- **Analytics integration**: User behavior tracking and insights
