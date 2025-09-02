# Ubuntu Server Deployment Guide

## Current Issue Solution

Based on your systemd status output, the service is failing to execute. Here's how to fix it:

### Quick Fix Commands

```bash
# 1. Stop the service
sudo systemctl stop mailserver

# 2. Check the binary
sudo ls -la /opt/mailserver/
sudo file /opt/mailserver/mserver

# 3. Fix permissions
sudo chmod +x /opt/mailserver/mserver
sudo chown -R mailserver:mailserver /opt/mailserver

# 4. Test the binary manually
cd /opt/mailserver
sudo -u mailserver ./mserver

# 5. If the binary runs, restart the service
sudo systemctl start mailserver
sudo systemctl status mailserver
```

### Complete Deployment Process

#### 1. Build for Linux (Run on Windows)

```powershell
# In PowerShell on Windows
.\build.ps1
```

This creates:
- `mserver-linux-amd64` (for most Ubuntu servers)
- `mserver-linux-arm64` (for ARM servers)

#### 2. Upload to Ubuntu Server

Upload these files to your Ubuntu server:
- `mserver-linux-amd64` (rename to `mserver`)
- `deploy.sh`
- `troubleshoot.sh`
- `static/` folder (if exists)

```bash
# Example using scp
scp mserver-linux-amd64 user@your-server:/tmp/mserver
scp deploy.sh user@your-server:/tmp/
scp troubleshoot.sh user@your-server:/tmp/
scp -r static user@your-server:/tmp/
```

#### 3. Deploy on Ubuntu Server

```bash
# SSH to your server
ssh user@your-server

# Make scripts executable
chmod +x /tmp/deploy.sh
chmod +x /tmp/troubleshoot.sh

# Run deployment
sudo /tmp/deploy.sh
```

#### 4. If Issues Occur

```bash
# Run troubleshooting script
sudo /tmp/troubleshoot.sh
```

### Manual Deployment Steps

If automated deployment fails, follow these manual steps:

```bash
# 1. Create user and directories
sudo useradd --system --home-dir /opt/mailserver --shell /bin/false mailserver
sudo mkdir -p /opt/mailserver/static

# 2. Copy files
sudo cp /tmp/mserver /opt/mailserver/
sudo cp -r /tmp/static/* /opt/mailserver/static/ 2>/dev/null || true

# 3. Set permissions
sudo chown -R mailserver:mailserver /opt/mailserver
sudo chmod +x /opt/mailserver/mserver

# 4. Create systemd service
sudo tee /etc/systemd/system/mailserver.service > /dev/null << EOF
[Unit]
Description=Email Server - Modern SMTP/IMAP Server with Web Interface
After=network.target
Wants=network.target

[Service]
Type=simple
User=mailserver
Group=mailserver
WorkingDirectory=/opt/mailserver
ExecStart=/opt/mailserver/mserver
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=mailserver

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable mailserver
sudo systemctl start mailserver

# 6. Check status
sudo systemctl status mailserver
```

### Firewall Configuration

```bash
# Allow required ports
sudo ufw allow 8080/tcp   # Web interface
sudo ufw allow 2525/tcp   # SMTP
sudo ufw allow 1143/tcp   # IMAP
sudo ufw reload
```

### Monitoring and Logs

```bash
# View live logs
sudo journalctl -u mailserver -f

# View recent logs
sudo journalctl -u mailserver -n 50

# Check service status
sudo systemctl status mailserver

# Restart service
sudo systemctl restart mailserver
```

### Common Issues and Solutions

#### 1. "Failed to execute" Error
- **Cause**: Binary built for wrong architecture or not executable
- **Solution**: 
  ```bash
  file /opt/mailserver/mserver
  chmod +x /opt/mailserver/mserver
  ```

#### 2. Permission Denied
- **Solution**:
  ```bash
  sudo chown -R mailserver:mailserver /opt/mailserver
  sudo chmod +x /opt/mailserver/mserver
  ```

#### 3. Port Already in Use
- **Check**: `sudo netstat -tulpn | grep -E ":8080|:2525|:1143"`
- **Solution**: Stop conflicting services or change ports in code

#### 4. Database Issues
- **Solution**: Ensure mailserver user can write to database file
  ```bash
  sudo chown mailserver:mailserver /opt/mailserver/email_server.db
  ```

### Accessing the Email Server

After successful deployment:

- **Web Interface**: `http://your-server-ip:8080`
- **SMTP Server**: `your-server-ip:2525`
- **IMAP Server**: `your-server-ip:1143`

### SSL/TLS Configuration (Optional)

For production use, consider:

1. **Reverse Proxy with Nginx**:
   ```nginx
   server {
       listen 80;
       server_name your-domain.com;
       location / {
           proxy_pass http://localhost:8080;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
       }
   }
   ```

2. **Let's Encrypt SSL**:
   ```bash
   sudo apt install certbot python3-certbot-nginx
   sudo certbot --nginx -d your-domain.com
   ```

### Backup and Maintenance

```bash
# Backup database
sudo cp /opt/mailserver/email_server.db /backup/email_server_$(date +%Y%m%d).db

# Update service
sudo systemctl stop mailserver
sudo cp new_mserver /opt/mailserver/mserver
sudo chmod +x /opt/mailserver/mserver
sudo chown mailserver:mailserver /opt/mailserver/mserver
sudo systemctl start mailserver
```
