#!/bin/bash

# Email Server Deployment Script for Ubuntu Server
# This script will deploy the email server as a systemd service

set -e

echo "Email Server Deployment Script"
echo "=============================="

# Configuration
SERVICE_NAME="mailserver"
SERVICE_USER="mailserver"
INSTALL_DIR="/opt/mailserver"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
BINARY_NAME="mserver"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run this script as root (use sudo)"
    exit 1
fi

# Create service user if it doesn't exist
if ! id "$SERVICE_USER" &>/dev/null; then
    echo "Creating service user: $SERVICE_USER"
    useradd --system --home-dir "$INSTALL_DIR" --shell /bin/false "$SERVICE_USER"
else
    echo "Service user $SERVICE_USER already exists"
fi

# Create installation directory
echo "Creating installation directory: $INSTALL_DIR"
mkdir -p "$INSTALL_DIR"
mkdir -p "$INSTALL_DIR/static"

# Stop service if running
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Stopping existing service..."
    systemctl stop "$SERVICE_NAME"
fi

# Copy binary and files
echo "Copying application files..."
cp "$BINARY_NAME" "$INSTALL_DIR/"
cp -r static/* "$INSTALL_DIR/static/" 2>/dev/null || echo "No static files found"

# Set permissions
echo "Setting permissions..."
chown -R "$SERVICE_USER:$SERVICE_USER" "$INSTALL_DIR"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Create systemd service file
echo "Creating systemd service file..."
cat > "$SERVICE_FILE" << EOF
[Unit]
Description=Email Server - Modern SMTP/IMAP Server with Web Interface
After=network.target
Wants=network.target

[Service]
Type=simple
User=$SERVICE_USER
Group=$SERVICE_USER
WorkingDirectory=$INSTALL_DIR
ExecStart=$INSTALL_DIR/$BINARY_NAME
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=$SERVICE_NAME

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR

# Environment
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd and enable service
echo "Reloading systemd configuration..."
systemctl daemon-reload

echo "Enabling service to start on boot..."
systemctl enable "$SERVICE_NAME"

# Start the service
echo "Starting service..."
systemctl start "$SERVICE_NAME"

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Service started successfully!"
    echo ""
    echo "Service Status:"
    systemctl status "$SERVICE_NAME" --no-pager
    echo ""
    echo "🌐 Web Interface: http://your-server-ip:8080"
    echo "📧 SMTP Server: your-server-ip:2525"
    echo "📬 IMAP Server: your-server-ip:1143"
    echo ""
    echo "📋 Useful commands:"
    echo "  sudo systemctl status $SERVICE_NAME    # Check status"
    echo "  sudo systemctl restart $SERVICE_NAME   # Restart service"
    echo "  sudo systemctl stop $SERVICE_NAME      # Stop service"
    echo "  sudo journalctl -u $SERVICE_NAME -f    # View logs"
else
    echo "❌ Service failed to start!"
    echo "Check the logs with: sudo journalctl -u $SERVICE_NAME -n 50"
    exit 1
fi
