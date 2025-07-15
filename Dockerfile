# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 nexus && \
    adduser -u 1001 -G nexus -s /bin/sh -D nexus

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/build/nexus /app/nexus

# Copy configuration file
COPY config.yaml /app/config.yaml

# Change ownership to nexus user
RUN chown -R nexus:nexus /app

# Switch to non-root user
USER nexus

# Expose port
EXPOSE 8080

# Set environment variables
ENV CONFIG_PATH=/app/config.yaml

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/nexus"]