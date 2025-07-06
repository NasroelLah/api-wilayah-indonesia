.PHONY: help run build clean test dev

# Default target
help:
	@echo "Available targets:"
	@echo "  run     - Run the application"
	@echo "  build   - Build the application"
	@echo "  clean   - Clean build artifacts"
	@echo "  test    - Run tests"
	@echo "  dev     - Run in development mode with hot reload"

# Run the application
run:
	go run main.go

# Build the application
build:
	go build -o wilayah-api main.go

# Clean build artifacts
clean:
	rm -f wilayah-api

# Run tests (add when tests are available)
test:
	go test ./...

# Development mode (if you want to add hot reload later)
dev:
	go run main.go

# Install dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Build for Windows
build-windows:
	GOOS=windows GOARCH=amd64 go build -o wilayah-api.exe main.go

# Build for Linux
build-linux:
	GOOS=linux GOARCH=amd64 go build -o wilayah-api-linux main.go

# Build for macOS
build-macos:
	GOOS=darwin GOARCH=amd64 go build -o wilayah-api-macos main.go

# Build all platforms
build-all: build-windows build-linux build-macos
