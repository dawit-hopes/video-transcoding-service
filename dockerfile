# --- Step 1: Build Stage ---
FROM golang:1.21-alpine AS builder

# Install build dependencies for CGO (required for Kafka)
RUN apk add --no-cache gcc musl-dev libc-dev pkgconfig pkgconf

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# CGO_ENABLED=1 is mandatory for the confluent-kafka-go library
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# --- Step 2: Runtime Stage ---
FROM alpine:3.18

# Install runtime dependencies: 
# 1. FFmpeg for transcoding
# 2. CA-certificates for secure connections
# 3. Libmusl/libc for CGO binary compatibility
RUN apk add --no-cache ffmpeg ca-certificates libc6-compat

WORKDIR /app

# Create necessary directories to match your volumes
RUN mkdir -p uploads output

# Copy the binary from the builder
COPY --from=builder /app/main .
# Copy index.html so the API can serve the gallery
COPY index.html .

# Set execution permissions
RUN chmod +x main

# The command is overridden in docker-compose, but we set a default
CMD ["./main", "-mode=all"]