# VS Code Debugging Setup

## Quick Start

1. Open the `API_Inventory_Management_System` folder in VS Code
2. Install the Go extension if not already installed
3. Set breakpoints in your Go code
4. Press `F5` to start debugging

## Debug Configuration

- **Name**: Debug Inventory Management System
- **Entry Point**: `packages/backend/services/inventory-management-system/cmd/server/main.go`
- **Environment**: Loads from `.env` file in the service directory
- **Working Directory**: Set to the inventory management system service directory

## Usage

1. Set breakpoints by clicking in the gutter next to line numbers
2. Use `F5` to start debugging
3. Use `F10` to step over, `F11` to step into functions
4. View variables in the Variables panel
5. Use the Debug Console to evaluate expressions

## Requirements

- Go extension for VS Code
- Go 1.22+ installed on your system
- Valid `.env` file in the service directory
