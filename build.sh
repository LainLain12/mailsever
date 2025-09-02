#!/bin/bash

# Build script for different architectures
# This script builds the email server for Linux deployment

echo "Building Email Server for Linux..."
echo "=================================="

# Build for Linux AMD64 (most common for servers)
echo "Building for Linux AMD64..."
GOOS=linux GOARCH=amd64 go build -o mserver-linux-amd64 .

# Build for Linux ARM64 (for ARM-based servers)
echo "Building for Linux ARM64..."
GOOS=linux GOARCH=arm64 go build -o mserver-linux-arm64 .

# Build for current platform (Windows)
echo "Building for current platform..."
go build -o mserver.exe .

echo ""
echo "Build completed!"
echo "Files created:"
echo "  mserver-linux-amd64  (for x86_64 Linux servers)"
echo "  mserver-linux-arm64  (for ARM64 Linux servers)"
echo "  mserver.exe          (for Windows)"
echo ""
echo "To deploy to Ubuntu server:"
echo "1. Upload mserver-linux-amd64 and rename it to 'mserver'"
echo "2. Upload the deploy.sh script"
echo "3. Run: chmod +x deploy.sh && sudo ./deploy.sh"
