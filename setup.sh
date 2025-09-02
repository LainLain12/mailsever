#!/bin/bash

echo "Email Server Setup Script"
echo "========================="

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Initialize go module if not exists
if [ ! -f "go.mod" ]; then
    go mod init email-server
fi

# Download dependencies
echo "Downloading dependencies..."
go mod tidy

# Build the application
echo "Building the application..."
go build -o email-server .

if [ $? -eq 0 ]; then
    echo "Build successful!"
    echo ""
    echo "To run the email server:"
    echo "  ./email-server (Linux/Mac)"
    echo "  email-server.exe (Windows)"
    echo ""
    echo "The server will start on:"
    echo "  Web interface: http://localhost:8080"
    echo "  SMTP server: localhost:2525"
    echo "  IMAP server: localhost:1143"
    echo ""
    echo "You can now create accounts and send/receive emails!"
else
    echo "Build failed. Please check the error messages above."
    exit 1
fi
