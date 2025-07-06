#!/bin/bash

echo "🚀 Starting Indonesian Region API..."
echo ""

# Check if JSON file exists
if [ ! -f "wilayah_final_2025.json" ]; then
    echo "❌ Error: wilayah_final_2025.json not found!"
    echo "Please ensure the JSON data file is in the current directory."
    exit 1
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed!"
    echo "Please install Go 1.21 or later."
    exit 1
fi

# Install dependencies
echo "📦 Installing dependencies..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ Failed to install dependencies"
    exit 1
fi

# Build the application
echo "🔨 Building application..."
go build -o wilayah-api main.go
if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful!"
echo ""

# Start the server in background
echo "🌐 Starting server on port 3000..."
./wilayah-api &
SERVER_PID=$!

# Wait for server to start
sleep 3

# Check if server is running
if ! curl -s http://localhost:3000/api/v1/health > /dev/null; then
    echo "❌ Server failed to start"
    kill $SERVER_PID 2>/dev/null
    exit 1
fi

echo "✅ Server is running!"
echo ""
echo "📚 API Documentation: http://localhost:3000/api/v1"
echo "🩺 Health Check: http://localhost:3000/api/v1/health"
echo ""
echo "🧪 Running manual tests..."
echo ""

# Run manual tests
go run test_manual.go

echo ""
echo "🛑 To stop the server, run: kill $SERVER_PID"
echo "Or press Ctrl+C and then run: kill $SERVER_PID"
echo ""
echo "📖 Examples:"
echo "  curl http://localhost:3000/api/v1/provinsi"
echo "  curl http://localhost:3000/api/v1/kabupaten?pro=73"
echo "  curl http://localhost:3000/api/v1/kecamatan?kec=7302"
echo "  curl http://localhost:3000/api/v1/desa?desa=7302010"
echo "  curl http://localhost:3000/api/v1/info/73"

# Keep server running
wait $SERVER_PID
