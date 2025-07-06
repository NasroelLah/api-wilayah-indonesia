# Use the official Golang image as base
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (required for some Go modules)
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wilayah-api main.go

# Use a minimal Alpine image for the final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/wilayah-api .

# Copy the JSON data file
COPY wilayah_final_2025.json .

# Expose port
EXPOSE 3000

# Set environment variable
ENV PORT=3000

# Run the application
CMD ["./wilayah-api"]
