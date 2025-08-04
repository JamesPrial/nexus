# Phase 4: Project Organization

## Prerequisites
- Phase 3 PR #18 should be merged first
- Ensure all tests pass and linting is clean

## Quick Start
```bash
cd /home/jamesprial/claude/nexus
git checkout main
git pull origin main  # Get Phase 3 changes
git checkout -b feat/phase4-organization
```

## Task 1: Separate overnight-tools
```bash
# Move to its own repository
cd /home/jamesprial/claude
mv nexus/overnight-tools overnight-tools-backup
git clone https://github.com/JamesPrial/nexus.git overnight-tools-temp
cd overnight-tools-temp
git filter-branch --subdirectory-filter overnight-tools -- --all
gh repo create overnight-tools --private --description "Overnight batch processing tools"
git remote set-url origin https://github.com/JamesPrial/overnight-tools.git
git push -u origin main

# Clean up nexus repo
cd ../nexus
git rm -rf overnight-tools
git commit -m "chore: move overnight-tools to separate repository

Moved overnight-tools to https://github.com/JamesPrial/overnight-tools
for better separation of concerns and independent versioning."
```

## Task 2: Create Comprehensive Documentation

### 2.1 Update README.md
Add sections for:
- Health check endpoint usage
- Metrics configuration and export
- Graceful shutdown behavior
- Integration test examples

### 2.2 Create Architecture Documentation
Create `docs/ARCHITECTURE.md`:
```markdown
# Nexus Architecture

## Overview
Nexus is a high-performance API gateway with clean dependency injection architecture.

## Request Flow
1. HTTP Request → Gateway Service
2. Validation Middleware (size, headers)
3. Authentication Middleware (API key validation)
4. Rate Limiting Middleware (per-client limits)
5. Token Limiting Middleware (AI token usage)
6. Proxy to Upstream API
7. Response with metrics collection

## Key Components
- **Container**: Central DI container managing dependencies
- **Gateway Service**: HTTP server with health endpoint
- **Middleware Chain**: Composable request processing
- **Metrics Collector**: Request and token usage tracking

## Health Monitoring
- `/health` endpoint for service status
- Metrics export via `/metrics` (when enabled)
- Graceful shutdown with connection draining
```

### 2.3 Create CHANGELOG.md
```markdown
# Changelog

## [Unreleased]
### Added
- Health endpoint at `/health`
- Metrics flush during graceful shutdown
- Comprehensive integration tests
- 73% test coverage for gateway package

### Fixed
- All golangci-lint issues resolved

## [0.1.0] - 2024-XX-XX
### Added
- Initial release
- API gateway with proxy functionality
- Rate limiting per client
- Token usage limiting
- Metrics collection and export
- Configurable authentication
```

## Task 3: Create First Release
```bash
# Ensure we're on main with all changes
cd /home/jamesprial/claude/nexus
git checkout main
git pull origin main

# Tag the release
git tag -a v0.1.0 -m "Release v0.1.0

Features:
- API gateway with clean architecture
- Rate limiting and token limiting
- Health endpoint for monitoring
- Metrics collection and export
- Graceful shutdown with connection draining
- 73% test coverage
- Comprehensive integration tests"

# Build release artifacts
make build-all

# Create GitHub release
gh release create v0.1.0 \
  --title "v0.1.0: Initial Release" \
  --notes "## 🎉 Initial Release

### Features
- 🚀 High-performance API gateway for AI services
- 🔒 API key authentication and transformation
- ⏱️ Rate limiting (requests per second)
- 🎯 Token usage limiting (per minute)
- 📊 Metrics collection and export
- 🏥 Health endpoint at /health
- 🔄 Graceful shutdown with connection draining
- ✅ 73% test coverage with integration tests

### Configuration
See config.yaml.example for configuration options.

### Docker Support
\`\`\`bash
docker build -t nexus .
docker run -p 8080:8080 -v ./config.yaml:/app/config.yaml nexus
\`\`\`

### Binary Downloads
Pre-built binaries are available below for Linux, macOS, and Windows." \
  build/nexus-*
```

## Task 4: Setup GitHub Actions
Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Install golangci-lint
        run: |
          curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.54.2
      
      - name: Run tests
        run: |
          go test -v -race -coverprofile=coverage.out ./...
          go tool cover -html=coverage.out -o coverage.html
      
      - name: Upload coverage
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html
      
      - name: Check coverage threshold
        run: |
          COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          echo "Total coverage: $COVERAGE%"
          if (( $(echo "$COVERAGE < 70" | bc -l) )); then
            echo "Coverage is below 70% threshold"
            exit 1
          fi
      
      - name: Run linter
        run: golangci-lint run ./...
```

Create `.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags:
      - 'v*'
      
jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      
      - name: Run tests
        run: go test -v ./...
        
      - name: Build all platforms
        run: |
          make build-all
          cd build && sha256sum nexus-* > checksums.txt
        
      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            build/nexus-*
            build/checksums.txt
          generate_release_notes: true
```

## Task 5: Improve Makefile
Add these targets to the Makefile:
```makefile
.PHONY: lint
lint:
	@if [ -f /tmp/golangci-lint ]; then \
		/tmp/golangci-lint run ./...; \
	else \
		golangci-lint run ./...; \
	fi

.PHONY: test-coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

.PHONY: bench
bench:
	go test -bench=. -benchmem ./...

.PHONY: install-tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Task 6: Future Improvements Document
Create `docs/FUTURE_IMPROVEMENTS.md`:
```markdown
# Future Improvements

Based on Phase 3 implementation experience:

## High Priority
1. **Metrics System Enhancement**
   - Ensure metrics collector is always initialized when enabled
   - Add metrics for health endpoint hits
   - Consider Prometheus native format

2. **Test Infrastructure**
   - Add benchmark tests for critical paths
   - Create test utilities for edge cases
   - Add load testing scenarios

3. **OpenTelemetry Integration**
   - Distributed tracing support
   - Correlation IDs for request tracking
   - Integration with popular APM tools

## Medium Priority
1. **Configuration Improvements**
   - Hot reload configuration
   - Configuration validation on startup
   - Environment-specific config files

2. **Operational Features**
   - Request/response logging options
   - Circuit breaker for upstream
   - Retry logic with exponential backoff

3. **Security Enhancements**
   - JWT authentication option
   - IP allowlisting/denylisting
   - Request signing validation

## Nice to Have
1. WebSocket support
2. gRPC gateway functionality
3. GraphQL support
4. Admin UI for metrics
```

## Success Criteria
- [ ] overnight-tools in separate repo
- [ ] v0.1.0 release published with binaries and checksums
- [ ] CI/CD pipeline working for tests and releases
- [ ] Comprehensive documentation including architecture
- [ ] Future improvements documented
- [ ] Makefile enhanced with useful targets

## Verification
```bash
# After completing all tasks
cd /home/jamesprial/claude/nexus

# Verify clean state
make test
make lint
make test-coverage  # Should show 70%+ coverage

# Test release process locally
git tag -a v0.1.0-test -m "Test release"
make build-all
ls -la build/  # Should have binaries for all platforms

# Clean up test tag
git tag -d v0.1.0-test
```

## Time Estimate
- Task 1 (Separate repos): 30 minutes
- Task 2 (Documentation): 45 minutes
- Task 3 (Release): 20 minutes
- Task 4 (GitHub Actions): 30 minutes
- Task 5 (Makefile): 15 minutes
- Task 6 (Future docs): 20 minutes
- Testing & Verification: 20 minutes

**Total: ~3 hours**