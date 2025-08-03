# Nexus API Gateway Makefile

# Build variables
BINARY_NAME=nexus
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S_UTC')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Directories
BUILD_DIR=build
DIST_DIR=dist

# Default target
.PHONY: all
all: clean test build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/gateway

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/gateway
	
	# Linux ARM64  
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/gateway
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/gateway
	
	# macOS ARM64 (Apple Silicon)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/gateway
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/gateway
	
	@echo "Cross-compilation complete. Binaries in $(DIST_DIR)/"

# Create release archives
.PHONY: release
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/releases
	
	# Linux AMD64
	tar -czf $(DIST_DIR)/releases/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-linux-amd64 -C ../. config.yaml README.md
	
	# Linux ARM64
	tar -czf $(DIST_DIR)/releases/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-linux-arm64 -C ../. config.yaml README.md
	
	# macOS AMD64
	tar -czf $(DIST_DIR)/releases/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-amd64 -C ../. config.yaml README.md
	
	# macOS ARM64
	tar -czf $(DIST_DIR)/releases/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-arm64 -C ../. config.yaml README.md
	
	# Windows AMD64
	cd $(DIST_DIR) && zip releases/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe ../config.yaml ../README.md
	
	@echo "Release archives created in $(DIST_DIR)/releases/"

# Install locally (copy to system PATH)
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete. Run '$(BINARY_NAME)' from anywhere."

# Uninstall from system
.PHONY: uninstall
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "Uninstall complete."

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	rm -f coverage.out coverage.html

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Development server (with hot reload)
.PHONY: dev
dev:
	@echo "Starting development server..."
	$(GOCMD) run ./cmd/gateway

# Run with production config
.PHONY: run
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Docker build
.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

# Run linter
.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found. Installing..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell go env GOPATH)/bin; \
	fi
	golangci-lint run --timeout=5m

# Run linter and automatically fix issues
.PHONY: lint-fix
lint-fix:
	@echo "Running golangci-lint with auto-fix..."
	golangci-lint run --fix --timeout=5m

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build binary for current platform"
	@echo "  build-all    - Build for all supported platforms"
	@echo "  release      - Create release archives"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  uninstall    - Remove binary from /usr/local/bin"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run golangci-lint"
	@echo "  lint-fix     - Run golangci-lint with auto-fix"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  dev          - Start development server"
	@echo "  run          - Build and run"
	@echo "  docker       - Build Docker image"
	@echo "  help         - Show this help"

# Show version info
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"