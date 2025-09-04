import React, { useState } from 'react';

const ProductCard = ({ product, cartQuantity, onAddToCart, onUpdateQuantity }) => {
  const [quantity, setQuantity] = useState(1);
  const [isAdding, setIsAdding] = useState(false);

  const handleAddToCart = async () => {
    if (quantity > 0 && quantity <= product.available) {
      setIsAdding(true);
      try {
        onAddToCart(product.product_id, quantity);
        setQuantity(1); // Reset quantity after adding
      } finally {
        setIsAdding(false);
      }
    }
  };

  const handleQuantityChange = (newQuantity) => {
    if (newQuantity >= 0 && newQuantity <= product.available) {
      setQuantity(newQuantity);
    }
  };

  const handleCartQuantityChange = (newQuantity) => {
    onUpdateQuantity(product.product_id, newQuantity);
  };

  const getStockStatus = () => {
    if (product.available === 0) return 'out-of-stock';
    if (product.available <= 5) return 'low-stock';
    return 'in-stock';
  };

  const getStockText = () => {
    if (product.available === 0) return 'Out of Stock';
    if (product.available <= 5) return `Low Stock (${product.available})`;
    return `${product.available} available`;
  };

  const formatPrice = (price) => {
    return `$${(price || 10).toFixed(2)}`;
  };

  const maxQuantity = Math.min(product.available, 99);
  const isOutOfStock = product.available === 0;
  const hasInCart = cartQuantity > 0;

  return (
    <div className={`product-card ${getStockStatus()}`}>
      <div className="product-header">
        <h3 className="product-id">{product.product_id}</h3>
        <div className={`stock-badge ${getStockStatus()}`}>
          {getStockText()}
        </div>
      </div>

      {product.name && (
        <div className="product-name">{product.name}</div>
      )}

      <div className="product-details">
        <div className="product-price">
          {formatPrice(product.price)}
        </div>
        
        {product.last_updated && (
          <div className="last-updated">
            Updated: {new Date(product.last_updated).toLocaleTimeString()}
          </div>
        )}
        
        {product.version && (
          <div className="version">
            v{product.version}
          </div>
        )}
      </div>

      {!isOutOfStock && (
        <div className="quantity-controls">
          <label>Quantity:</label>
          <div className="quantity-input">
            <button
              onClick={() => handleQuantityChange(quantity - 1)}
              disabled={quantity <= 1}
              className="quantity-btn"
            >
              -
            </button>
            <input
              type="number"
              min="1"
              max={maxQuantity}
              value={quantity}
              onChange={(e) => handleQuantityChange(parseInt(e.target.value) || 1)}
              className="quantity-field"
            />
            <button
              onClick={() => handleQuantityChange(quantity + 1)}
              disabled={quantity >= maxQuantity}
              className="quantity-btn"
            >
              +
            </button>
          </div>
        </div>
      )}

      <div className="product-actions">
        {!isOutOfStock && (
          <button
            onClick={handleAddToCart}
            disabled={isAdding || quantity > product.available}
            className="add-to-cart-btn"
          >
            {isAdding ? '⏳ Adding...' : `🛒 Add ${quantity} to Cart`}
          </button>
        )}

        {hasInCart && (
          <div className="cart-controls">
            <div className="in-cart-label">
              In cart: {cartQuantity}
            </div>
            <div className="cart-quantity-controls">
              <button
                onClick={() => handleCartQuantityChange(cartQuantity - 1)}
                className="cart-btn decrease"
                title="Remove one from cart"
              >
                -
              </button>
              <span className="cart-quantity">{cartQuantity}</span>
              <button
                onClick={() => handleCartQuantityChange(cartQuantity + 1)}
                disabled={cartQuantity >= product.available}
                className="cart-btn increase"
                title="Add one more to cart"
              >
                +
              </button>
              <button
                onClick={() => handleCartQuantityChange(0)}
                className="cart-btn remove"
                title="Remove all from cart"
              >
                🗑️
              </button>
            </div>
          </div>
        )}
      </div>

      {isOutOfStock && (
        <div className="out-of-stock-message">
          This item is currently out of stock
        </div>
      )}
    </div>
  );
};

export default ProductCard;
