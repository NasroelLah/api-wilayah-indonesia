@echo off
echo 🚀 Starting Indonesian Region API...
echo.

REM Check if JSON file exists
if not exist "wilayah_final_2025.json" (
    echo ❌ Error: wilayah_final_2025.json not found!
    echo Please ensure the JSON data file is in the current directory.
    pause
    exit /b 1
)

REM Check if Go is installed
go version >nul 2>&1
if errorlevel 1 (
    echo ❌ Error: Go is not installed!
    echo Please install Go 1.21 or later.
    pause
    exit /b 1
)

REM Install dependencies
echo 📦 Installing dependencies...
go mod tidy
if errorlevel 1 (
    echo ❌ Failed to install dependencies
    pause
    exit /b 1
)

REM Build the application
echo 🔨 Building application...
go build -o wilayah-api.exe main.go
if errorlevel 1 (
    echo ❌ Build failed
    pause
    exit /b 1
)

echo ✅ Build successful!
echo.

REM Start the server
echo 🌐 Starting server on port 3000...
echo.
echo 📚 API Documentation: http://localhost:3000/api/v1
echo 🩺 Health Check: http://localhost:3000/api/v1/health
echo.
echo 📖 Examples:
echo   curl http://localhost:3000/api/v1/provinsi
echo   curl http://localhost:3000/api/v1/kabupaten?pro=73
echo   curl "http://localhost:3000/api/v1/kecamatan?kec=7302"
echo   curl "http://localhost:3000/api/v1/desa?desa=7302010"
echo   curl http://localhost:3000/api/v1/info/73
echo.
echo Press Ctrl+C to stop the server
echo.

wilayah-api.exe
