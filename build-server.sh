#!/bin/bash

# Build script for Ubuntu server
# This script installs dependencies and builds the email server

echo "Installing build dependencies..."
sudo apt update
sudo apt install -y build-essential gcc

echo "Building email server..."
go clean -cache
go mod tidy
go build -v -o mserver .

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "Run: sudo ./deploy.sh to deploy the server"
else
    echo "❌ Build failed!"
    exit 1
fi
