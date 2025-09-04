import React, { useState, useEffect } from 'react';
import ProductList from './components/ProductList';
import Cart from './components/Cart';
import Header from './components/Header';
import { getProducts, updateInventory } from './services/api';
import './App.css';

function App() {
  const [products, setProducts] = useState([]);
  const [cart, setCart] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [purchasing, setPurchasing] = useState(false);
  const [message, setMessage] = useState(null);

  // Load products on component mount
  useEffect(() => {
    loadProducts();
    
    // Set up auto-refresh every 30 seconds
    const interval = setInterval(loadProducts, 30000);
    return () => clearInterval(interval);
  }, []);

  const loadProducts = async () => {
    try {
      setError(null);
      const data = await getProducts();
      setProducts(data || []);
    } catch (err) {
      setError('Failed to load products: ' + err.message);
      console.error('Error loading products:', err);
    } finally {
      setLoading(false);
    }
  };

  const addToCart = (productId, quantity) => {
    setCart(prev => ({
      ...prev,
      [productId]: (prev[productId] || 0) + quantity
    }));
  };

  const updateCartQuantity = (productId, quantity) => {
    if (quantity <= 0) {
      const newCart = { ...cart };
      delete newCart[productId];
      setCart(newCart);
    } else {
      setCart(prev => ({
        ...prev,
        [productId]: quantity
      }));
    }
  };

  const clearCart = () => {
    setCart({});
  };

  const processPurchase = async () => {
    if (Object.keys(cart).length === 0) {
      setMessage({ type: 'error', text: 'Cart is empty' });
      return;
    }

    setPurchasing(true);
    setMessage(null);

    try {
      // Process each item in cart
      const results = [];
      for (const [productId, quantity] of Object.entries(cart)) {
        try {
          const result = await updateInventory(productId, -quantity);
          results.push({ productId, success: true, result });
        } catch (err) {
          results.push({ productId, success: false, error: err.message });
        }
      }

      // Check results
      const successful = results.filter(r => r.success);
      const failed = results.filter(r => !r.success);

      if (successful.length > 0) {
        setMessage({
          type: 'success',
          text: `Successfully purchased ${successful.length} item(s)${
            failed.length > 0 ? `, ${failed.length} failed` : ''
          }`
        });
        
        // Clear successful items from cart
        const newCart = { ...cart };
        successful.forEach(({ productId }) => {
          delete newCart[productId];
        });
        setCart(newCart);
        
        // Refresh products to show updated quantities
        await loadProducts();
      } else {
        setMessage({
          type: 'error',
          text: 'All purchases failed. Please check inventory levels.'
        });
      }
    } catch (err) {
      setMessage({
        type: 'error',
        text: 'Purchase failed: ' + err.message
      });
    } finally {
      setPurchasing(false);
    }
  };

  const getTotalItems = () => {
    return Object.values(cart).reduce((sum, qty) => sum + qty, 0);
  };

  const getTotalValue = () => {
    return Object.entries(cart).reduce((sum, [productId, qty]) => {
      const product = products.find(p => p.product_id === productId);
      return sum + (product ? (product.price || 10) * qty : 0);
    }, 0);
  };

  if (loading) {
    return (
      <div className="loading">
        <div className="spinner"></div>
        Loading products...
      </div>
    );
  }

  return (
    <div className="app">
      <Header 
        totalItems={getTotalItems()}
        onRefresh={loadProducts}
        loading={loading}
      />
      
      {error && (
        <div className="error-banner">
          {error}
          <button onClick={loadProducts} className="retry-btn">
            Retry
          </button>
        </div>
      )}

      {message && (
        <div className={`message ${message.type}`}>
          {message.text}
          <button onClick={() => setMessage(null)} className="close-btn">
            ×
          </button>
        </div>
      )}

      <div className="main-content">
        <div className="products-section">
          <ProductList 
            products={products}
            cart={cart}
            onAddToCart={addToCart}
            onUpdateQuantity={updateCartQuantity}
          />
        </div>
        
        <div className="cart-section">
          <Cart
            cart={cart}
            products={products}
            totalItems={getTotalItems()}
            totalValue={getTotalValue()}
            onUpdateQuantity={updateCartQuantity}
            onClearCart={clearCart}
            onPurchase={processPurchase}
            purchasing={purchasing}
          />
        </div>
      </div>
    </div>
  );
}

export default App;
