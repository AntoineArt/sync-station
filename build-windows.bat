@echo off
echo ========================================
echo Config Sync Tool - Windows Build Script
echo ========================================
echo.

echo Checking Go installation...
go version >nul 2>&1
if errorlevel 1 (
    echo ERROR: Go is not installed or not in PATH
    echo Please install Go from https://golang.org/dl/
    pause
    exit /b 1
)

echo Go found! Proceeding with build...
echo.

echo Updating dependencies...
go mod tidy

echo.
echo Building for Windows (64-bit)...
go build -o config-sync-tool-windows.exe .
if errorlevel 1 (
    echo ERROR: Build failed for Windows 64-bit
    pause
    exit /b 1
)

echo.
echo Building for Windows (32-bit)...
set GOOS=windows
set GOARCH=386
go build -o config-sync-tool-windows-32.exe .
if errorlevel 1 (
    echo WARNING: Build failed for Windows 32-bit (continuing...)
)

echo.
echo ========================================
echo Build Complete!
echo ========================================
echo.
echo Files created:
if exist config-sync-tool-windows.exe (
    echo   ✓ config-sync-tool-windows.exe (64-bit)
    for %%I in (config-sync-tool-windows.exe) do echo     Size: %%~zI bytes
)
if exist config-sync-tool-windows-32.exe (
    echo   ✓ config-sync-tool-windows-32.exe (32-bit)
    for %%I in (config-sync-tool-windows-32.exe) do echo     Size: %%~zI bytes
)
echo.
echo To run the application, double-click config-sync-tool-windows.exe
echo or run it from command prompt.
echo.
pause