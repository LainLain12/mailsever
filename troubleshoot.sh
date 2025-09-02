#!/bin/bash

# Troubleshooting script for mailserver service issues
# Run this script to diagnose and fix common deployment problems

echo "Email Server Troubleshooting Script"
echo "===================================="

SERVICE_NAME="mailserver"
INSTALL_DIR="/opt/mailserver"
BINARY_NAME="mserver"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Please run this script as root (use sudo)"
    exit 1
fi

echo "1. Checking service status..."
systemctl status "$SERVICE_NAME" --no-pager || echo "Service not found or not running"

echo ""
echo "2. Checking binary file..."
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo "✅ Binary exists: $INSTALL_DIR/$BINARY_NAME"
    ls -la "$INSTALL_DIR/$BINARY_NAME"
    
    # Check if it's executable
    if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo "✅ Binary is executable"
    else
        echo "❌ Binary is not executable - fixing..."
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Check architecture
    echo "Binary info:"
    file "$INSTALL_DIR/$BINARY_NAME"
    
    # Test if binary can run
    echo ""
    echo "Testing binary execution..."
    cd "$INSTALL_DIR"
    timeout 5s ./"$BINARY_NAME" --help 2>&1 || echo "Binary test completed"
    
else
    echo "❌ Binary not found at $INSTALL_DIR/$BINARY_NAME"
    echo "Please ensure you've uploaded the correct binary file"
fi

echo ""
echo "3. Checking permissions..."
ls -la "$INSTALL_DIR/"

echo ""
echo "4. Checking user..."
if id mailserver &>/dev/null; then
    echo "✅ Service user 'mailserver' exists"
    id mailserver
else
    echo "❌ Service user 'mailserver' not found - creating..."
    useradd --system --home-dir "$INSTALL_DIR" --shell /bin/false mailserver
    chown -R mailserver:mailserver "$INSTALL_DIR"
fi

echo ""
echo "5. Checking ports..."
echo "Checking if required ports are available..."
netstat -tulpn | grep -E ":8080|:2525|:1143" || echo "Ports appear to be free"

echo ""
echo "6. Checking systemd service file..."
if [ -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
    echo "✅ Service file exists"
    echo "Service file content:"
    cat "/etc/systemd/system/$SERVICE_NAME.service"
else
    echo "❌ Service file not found - creating..."
    cat > "/etc/systemd/system/$SERVICE_NAME.service" << EOF
[Unit]
Description=Email Server - Modern SMTP/IMAP Server with Web Interface
After=network.target
Wants=network.target

[Service]
Type=simple
User=mailserver
Group=mailserver
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

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
fi

echo ""
echo "7. Recent logs..."
echo "Last 20 log entries:"
journalctl -u "$SERVICE_NAME" -n 20 --no-pager || echo "No logs found"

echo ""
echo "8. System information..."
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d'"' -f2)"
echo "Architecture: $(uname -m)"
echo "Kernel: $(uname -r)"

echo ""
echo "9. Quick fixes..."
echo "Applying common fixes..."

# Fix permissions
chown -R mailserver:mailserver "$INSTALL_DIR"
chmod +x "$INSTALL_DIR/$BINARY_NAME"

# Reload and restart
systemctl daemon-reload
systemctl stop "$SERVICE_NAME" 2>/dev/null || true
sleep 2
systemctl start "$SERVICE_NAME"

echo ""
echo "10. Final status check..."
sleep 3
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Service is now running!"
    systemctl status "$SERVICE_NAME" --no-pager
    echo ""
    echo "🌐 Web Interface: http://$(hostname -I | awk '{print $1}'):8080"
else
    echo "❌ Service still not running"
    echo "Recent error logs:"
    journalctl -u "$SERVICE_NAME" -n 10 --no-pager
    echo ""
    echo "Common solutions:"
    echo "1. Check if the binary is built for the correct architecture"
    echo "2. Ensure all dependencies are available"
    echo "3. Check firewall settings"
    echo "4. Verify the binary has proper permissions"
fi
