import React, { useState } from 'react';
import ProductCard from './ProductCard';

const ProductList = ({ products, cart, onAddToCart, onUpdateQuantity }) => {
  const [currentPage, setCurrentPage] = useState(1);
  const [searchTerm, setSearchTerm] = useState('');
  const [sortBy, setSortBy] = useState('product_id');
  const [sortOrder, setSortOrder] = useState('asc');
  
  const itemsPerPage = 12;

  // Filter products based on search term
  const filteredProducts = products.filter(product =>
    product.product_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
    (product.name && product.name.toLowerCase().includes(searchTerm.toLowerCase()))
  );

  // Sort products
  const sortedProducts = [...filteredProducts].sort((a, b) => {
    let aValue = a[sortBy];
    let bValue = b[sortBy];
    
    // Handle numeric sorting
    if (sortBy === 'available' || sortBy === 'price') {
      aValue = Number(aValue) || 0;
      bValue = Number(bValue) || 0;
    }
    
    // Handle string sorting
    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase();
      bValue = bValue.toLowerCase();
    }
    
    if (sortOrder === 'asc') {
      return aValue > bValue ? 1 : -1;
    } else {
      return aValue < bValue ? 1 : -1;
    }
  });

  // Paginate products
  const totalPages = Math.ceil(sortedProducts.length / itemsPerPage);
  const startIndex = (currentPage - 1) * itemsPerPage;
  const paginatedProducts = sortedProducts.slice(startIndex, startIndex + itemsPerPage);

  const handleSort = (field) => {
    if (sortBy === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortBy(field);
      setSortOrder('asc');
    }
  };

  const getSortIcon = (field) => {
    if (sortBy !== field) return '↕️';
    return sortOrder === 'asc' ? '↑' : '↓';
  };

  if (products.length === 0) {
    return (
      <div className="product-list">
        <div className="empty-state">
          <h3>No products available</h3>
          <p>The store inventory is empty or still loading.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="product-list">
      <div className="product-list-header">
        <h2>Available Products ({filteredProducts.length})</h2>
        
        <div className="product-controls">
          <div className="search-box">
            <input
              type="text"
              placeholder="Search products..."
              value={searchTerm}
              onChange={(e) => {
                setSearchTerm(e.target.value);
                setCurrentPage(1); // Reset to first page when searching
              }}
              className="search-input"
            />
            <span className="search-icon">🔍</span>
          </div>
          
          <div className="sort-controls">
            <label>Sort by:</label>
            <button 
              onClick={() => handleSort('product_id')}
              className={`sort-btn ${sortBy === 'product_id' ? 'active' : ''}`}
            >
              ID {getSortIcon('product_id')}
            </button>
            <button 
              onClick={() => handleSort('available')}
              className={`sort-btn ${sortBy === 'available' ? 'active' : ''}`}
            >
              Stock {getSortIcon('available')}
            </button>
          </div>
        </div>
      </div>

      {filteredProducts.length === 0 ? (
        <div className="empty-state">
          <h3>No products found</h3>
          <p>Try adjusting your search terms.</p>
        </div>
      ) : (
        <>
          <div className="product-grid">
            {paginatedProducts.map(product => (
              <ProductCard
                key={product.product_id}
                product={product}
                cartQuantity={cart[product.product_id] || 0}
                onAddToCart={onAddToCart}
                onUpdateQuantity={onUpdateQuantity}
              />
            ))}
          </div>

          {totalPages > 1 && (
            <div className="pagination">
              <button
                onClick={() => setCurrentPage(prev => Math.max(1, prev - 1))}
                disabled={currentPage === 1}
                className="pagination-btn"
              >
                ← Previous
              </button>
              
              <div className="pagination-info">
                <span>
                  Page {currentPage} of {totalPages}
                </span>
                <span className="pagination-details">
                  ({startIndex + 1}-{Math.min(startIndex + itemsPerPage, filteredProducts.length)} of {filteredProducts.length})
                </span>
              </div>
              
              <button
                onClick={() => setCurrentPage(prev => Math.min(totalPages, prev + 1))}
                disabled={currentPage === totalPages}
                className="pagination-btn"
              >
                Next →
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
};

export default ProductList;
