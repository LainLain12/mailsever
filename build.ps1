# Build script for Windows
# This script builds the email server for different platforms

Write-Host "Building Email Server for Multiple Platforms..." -ForegroundColor Green
Write-Host "===============================================" -ForegroundColor Green

# Build for Linux AMD64 (most common for servers)
Write-Host "Building for Linux AMD64..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o mserver-linux-amd64 .

# Build for Linux ARM64 (for ARM-based servers)
Write-Host "Building for Linux ARM64..." -ForegroundColor Yellow
$env:GOOS = "linux"
$env:GOARCH = "arm64"
go build -o mserver-linux-arm64 .

# Reset environment variables and build for Windows
Write-Host "Building for Windows..." -ForegroundColor Yellow
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
go build -o mserver.exe .

Write-Host ""
Write-Host "Build completed!" -ForegroundColor Green
Write-Host "Files created:" -ForegroundColor Cyan
Write-Host "  mserver-linux-amd64  (for x86_64 Linux servers)" -ForegroundColor White
Write-Host "  mserver-linux-arm64  (for ARM64 Linux servers)" -ForegroundColor White
Write-Host "  mserver.exe          (for Windows)" -ForegroundColor White
Write-Host ""
Write-Host "To deploy to Ubuntu server:" -ForegroundColor Yellow
Write-Host "1. Upload mserver-linux-amd64 and rename it to 'mserver'" -ForegroundColor White
Write-Host "2. Upload the deploy.sh script" -ForegroundColor White
Write-Host "3. Run: chmod +x deploy.sh && sudo ./deploy.sh" -ForegroundColor White
