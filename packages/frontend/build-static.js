#!/usr/bin/env node

// Custom build script to process static HTML with environment variables

const fs = require('fs');
const path = require('path');

console.log('🔧 Building static frontend with environment variables...');

// Create build directory
const buildDir = path.join(__dirname, 'build');
if (!fs.existsSync(buildDir)) {
  fs.mkdirSync(buildDir, { recursive: true });
}

// Copy public directory to build
const publicDir = path.join(__dirname, 'public');
const copyRecursive = (src, dest) => {
  const stats = fs.statSync(src);
  if (stats.isDirectory()) {
    if (!fs.existsSync(dest)) {
      fs.mkdirSync(dest, { recursive: true });
    }
    const files = fs.readdirSync(src);
    files.forEach(file => {
      copyRecursive(path.join(src, file), path.join(dest, file));
    });
  } else {
    fs.copyFileSync(src, dest);
  }
};

copyRecursive(publicDir, buildDir);

// Process HTML files to replace environment variables
const processFile = (filePath) => {
  if (path.extname(filePath) === '.html') {
    console.log(`Processing: ${filePath}`);
    let content = fs.readFileSync(filePath, 'utf8');
    
    // Replace environment variable placeholders with actual values
    const envVars = {
      'REACT_APP_STORE_NAME': process.env.REACT_APP_STORE_NAME || 'Store S1',
      'REACT_APP_STORE_ID': process.env.REACT_APP_STORE_ID || 'store-s1',
      'REACT_APP_API_KEY': process.env.REACT_APP_API_KEY || 'demo',
      'REACT_APP_STORE_API_URL': process.env.REACT_APP_STORE_API_URL || 'http://store-s1:8083',
      'REACT_APP_API_BASE_URL': process.env.REACT_APP_API_BASE_URL || '/api',
      'REACT_APP_AUTO_REFRESH_INTERVAL': process.env.REACT_APP_AUTO_REFRESH_INTERVAL || '30000',
      'REACT_APP_REQUEST_TIMEOUT': process.env.REACT_APP_REQUEST_TIMEOUT || '10000'
    };
    
    // Replace placeholders
    Object.entries(envVars).forEach(([key, value]) => {
      const placeholder = `%${key}%`;
      content = content.replace(new RegExp(placeholder, 'g'), value);
    });
    
    // Also replace the JavaScript constants
    content = content.replace(
      /const STORE_NAME = '[^']*'/g,
      `const STORE_NAME = '${envVars.REACT_APP_STORE_NAME}'`
    );
    content = content.replace(
      /const STORE_ID = '[^']*'/g,
      `const STORE_ID = '${envVars.REACT_APP_STORE_ID}'`
    );
    content = content.replace(
      /const API_KEY = '[^']*'/g,
      `const API_KEY = '${envVars.REACT_APP_API_KEY}'`
    );
    
    fs.writeFileSync(filePath, content);
    console.log(`✅ Processed: ${filePath}`);
  }
};

// Process all HTML files in build directory
const processDirectory = (dir) => {
  const files = fs.readdirSync(dir);
  files.forEach(file => {
    const filePath = path.join(dir, file);
    const stats = fs.statSync(filePath);
    if (stats.isDirectory()) {
      processDirectory(filePath);
    } else {
      processFile(filePath);
    }
  });
};

processDirectory(buildDir);

console.log('🎉 Build completed successfully!');
console.log('📋 Environment variables used:');
console.log(`  REACT_APP_STORE_NAME: ${process.env.REACT_APP_STORE_NAME || 'Store S1'}`);
console.log(`  REACT_APP_STORE_ID: ${process.env.REACT_APP_STORE_ID || 'store-s1'}`);
console.log(`  REACT_APP_API_KEY: ${process.env.REACT_APP_API_KEY || 'demo'}`);
