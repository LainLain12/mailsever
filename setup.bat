@echo off
echo Email Server Setup Script
echo =========================

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo Go is not installed. Please install Go 1.21 or later.
    pause
    exit /b 1
)

REM Download dependencies
echo Downloading dependencies...
go mod tidy

REM Build the application
echo Building the application...
go build -o email-server.exe .

if %errorlevel% equ 0 (
    echo Build successful!
    echo.
    echo To run the email server:
    echo   email-server.exe
    echo.
    echo The server will start on:
    echo   Web interface: http://localhost:8080
    echo   SMTP server: localhost:2525
    echo   IMAP server: localhost:1143
    echo.
    echo You can now create accounts and send/receive emails!
    pause
) else (
    echo Build failed. Please check the error messages above.
    pause
    exit /b 1
)
