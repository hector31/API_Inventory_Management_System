import React, { useState, useEffect } from 'react';
import { getSyncStatus, forceSync } from '../services/api';

const Header = ({ totalItems, onRefresh, loading }) => {
  const [syncStatus, setSyncStatus] = useState(null);
  const [syncing, setSyncing] = useState(false);
  const [lastUpdate, setLastUpdate] = useState(new Date());

  useEffect(() => {
    loadSyncStatus();
    const interval = setInterval(loadSyncStatus, 10000); // Check every 10 seconds
    return () => clearInterval(interval);
  }, []);

  const loadSyncStatus = async () => {
    try {
      const status = await getSyncStatus();
      setSyncStatus(status);
      setLastUpdate(new Date());
    } catch (error) {
      console.error('Failed to load sync status:', error);
    }
  };

  const handleForceSync = async () => {
    setSyncing(true);
    try {
      await forceSync();
      await loadSyncStatus();
      onRefresh(); // Refresh products after sync
    } catch (error) {
      console.error('Failed to force sync:', error);
    } finally {
      setSyncing(false);
    }
  };

  const formatTime = (date) => {
    return date.toLocaleTimeString();
  };

  return (
    <header className="header">
      <div className="header-content">
        <div className="header-left">
          <h1>🏪 Store S1 Inventory</h1>
          <div className="sync-info">
            {syncStatus && (
              <div className={`sync-status ${syncStatus.last_sync_success ? 'success' : 'error'}`}>
                <span className="sync-indicator">
                  {syncStatus.in_progress ? '🔄' : syncStatus.last_sync_success ? '✅' : '❌'}
                </span>
                <span className="sync-text">
                  {syncStatus.in_progress 
                    ? 'Syncing...' 
                    : syncStatus.last_sync_success 
                      ? `${syncStatus.product_count} products` 
                      : 'Sync failed'
                  }
                </span>
                {syncStatus.last_sync_time && (
                  <span className="sync-time">
                    Last: {new Date(syncStatus.last_sync_time).toLocaleTimeString()}
                  </span>
                )}
              </div>
            )}
          </div>
        </div>

        <div className="header-center">
          <div className="cart-summary">
            <span className="cart-icon">🛒</span>
            <span className="cart-count">{totalItems}</span>
            <span className="cart-text">items in cart</span>
          </div>
        </div>

        <div className="header-right">
          <div className="header-actions">
            <button 
              onClick={onRefresh} 
              disabled={loading}
              className="action-btn refresh-btn"
              title="Refresh products"
            >
              {loading ? '🔄' : '↻'} Refresh
            </button>
            
            <button 
              onClick={handleForceSync} 
              disabled={syncing}
              className="action-btn sync-btn"
              title="Force sync with central API"
            >
              {syncing ? '🔄' : '🔄'} Sync
            </button>
          </div>
          
          <div className="last-update">
            Updated: {formatTime(lastUpdate)}
          </div>
        </div>
      </div>
    </header>
  );
};

export default Header;
