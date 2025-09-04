import React from 'react';

const Cart = ({ 
  cart, 
  products, 
  totalItems, 
  totalValue, 
  onUpdateQuantity, 
  onClearCart, 
  onPurchase, 
  purchasing 
}) => {
  const cartItems = Object.entries(cart).map(([productId, quantity]) => {
    const product = products.find(p => p.product_id === productId);
    return {
      productId,
      quantity,
      product: product || { product_id: productId, available: 0, price: 10 }
    };
  });

  const isEmpty = cartItems.length === 0;

  const formatPrice = (price) => {
    return `$${(price || 10).toFixed(2)}`;
  };

  const getItemTotal = (item) => {
    return (item.product.price || 10) * item.quantity;
  };

  const canPurchase = () => {
    if (isEmpty || purchasing) return false;
    
    // Check if all items have sufficient stock
    return cartItems.every(item => 
      item.quantity <= item.product.available
    );
  };

  const getStockWarnings = () => {
    return cartItems.filter(item => 
      item.quantity > item.product.available
    );
  };

  const stockWarnings = getStockWarnings();

  return (
    <div className="cart">
      <div className="cart-header">
        <h2>🛒 Shopping Cart</h2>
        {!isEmpty && (
          <button 
            onClick={onClearCart}
            className="clear-cart-btn"
            disabled={purchasing}
          >
            Clear All
          </button>
        )}
      </div>

      {isEmpty ? (
        <div className="empty-cart">
          <div className="empty-cart-icon">🛒</div>
          <h3>Your cart is empty</h3>
          <p>Add some products to get started!</p>
        </div>
      ) : (
        <>
          <div className="cart-items">
            {cartItems.map(item => (
              <div key={item.productId} className="cart-item">
                <div className="cart-item-header">
                  <h4 className="cart-item-id">{item.productId}</h4>
                  <button
                    onClick={() => onUpdateQuantity(item.productId, 0)}
                    className="remove-item-btn"
                    disabled={purchasing}
                    title="Remove from cart"
                  >
                    ×
                  </button>
                </div>

                <div className="cart-item-details">
                  <div className="cart-item-price">
                    {formatPrice(item.product.price)} each
                  </div>
                  
                  <div className="cart-item-quantity">
                    <button
                      onClick={() => onUpdateQuantity(item.productId, item.quantity - 1)}
                      disabled={purchasing || item.quantity <= 1}
                      className="quantity-btn"
                    >
                      -
                    </button>
                    <span className="quantity-display">{item.quantity}</span>
                    <button
                      onClick={() => onUpdateQuantity(item.productId, item.quantity + 1)}
                      disabled={purchasing || item.quantity >= item.product.available}
                      className="quantity-btn"
                    >
                      +
                    </button>
                  </div>

                  <div className="cart-item-total">
                    {formatPrice(getItemTotal(item))}
                  </div>
                </div>

                <div className="cart-item-stock">
                  <span className={`stock-info ${item.quantity > item.product.available ? 'insufficient' : 'sufficient'}`}>
                    {item.product.available} available
                    {item.quantity > item.product.available && (
                      <span className="stock-warning"> - Insufficient stock!</span>
                    )}
                  </span>
                </div>
              </div>
            ))}
          </div>

          {stockWarnings.length > 0 && (
            <div className="stock-warnings">
              <h4>⚠️ Stock Warnings</h4>
              {stockWarnings.map(item => (
                <div key={item.productId} className="stock-warning-item">
                  <strong>{item.productId}</strong>: 
                  Requested {item.quantity}, but only {item.product.available} available
                </div>
              ))}
            </div>
          )}

          <div className="cart-summary">
            <div className="cart-totals">
              <div className="total-items">
                Total Items: <strong>{totalItems}</strong>
              </div>
              <div className="total-value">
                Total Value: <strong>{formatPrice(totalValue)}</strong>
              </div>
            </div>

            <div className="cart-actions">
              <button
                onClick={onPurchase}
                disabled={!canPurchase()}
                className={`purchase-btn ${canPurchase() ? 'enabled' : 'disabled'}`}
              >
                {purchasing ? (
                  <>
                    <span className="spinner-small"></span>
                    Processing...
                  </>
                ) : (
                  `💳 Purchase ${totalItems} item${totalItems !== 1 ? 's' : ''}`
                )}
              </button>

              {!canPurchase() && !purchasing && !isEmpty && (
                <div className="purchase-disabled-reason">
                  {stockWarnings.length > 0 
                    ? 'Insufficient stock for some items'
                    : 'Cannot process purchase'
                  }
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
};

export default Cart;
