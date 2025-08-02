# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 🚀 Agent-First Development

**IMPORTANT**: All code development MUST use the specialized agent system. Agents enforce TDD and maintain code quality standards.

### Setting Up Agents

```bash
# Agents are in the agents/ directory
# Create symlink for auto-discovery (if not exists)
mkdir -p .claude
ln -s ../agents .claude/agents
```

### Primary Development Workflow

**ALL feature development follows this MANDATORY flow:**

```
1. Create feature branch → git checkout -b feat/feature-name
2. nexus-test-designer → Creates comprehensive failing tests (RED)
3. nexus-rapid-impl → Makes tests pass with minimal code (GREEN)  
4. nexus-perf-optimizer → Optimizes if performance benchmarks fail
5. code-refactor → Improves code quality (REFACTOR)
6. nexus-integration-tester → Validates end-to-end functionality
7. Push branch → git push -u origin feat/feature-name
8. Create PR → Use GitHub CLI or web interface
```

**Example Usage:**
```bash
# Start EVERY feature with a new branch:
git checkout -b feat/jwt-authentication

# Then begin TDD cycle:
"Use nexus-test-designer to create tests for JWT authentication"

# After all tests pass and code is clean:
"Use nexus-integration-tester to validate JWT auth through full middleware chain"

# When integration tests pass:
git add .
git commit -m "feat: add JWT authentication with comprehensive tests"
git push -u origin feat/jwt-authentication

# Create PR for review
gh pr create --title "Add JWT authentication" --body "Implements JWT auth with full test coverage"
```

## 🏗️ Architecture Overview

### Core Design Philosophy

Nexus uses **interface-driven dependency injection** with clean separation of concerns. Every component is testable, mockable, and replaceable.

### Request Flow Architecture

```
Client Request → Validation → Authentication → Rate Limiting → Token Limiting → Proxy → Upstream
                     ↓             ↓               ↓              ↓            ↓
                  400/413       401/403          429           429          Response
```

### Key Components

1. **Container** (`internal/container/container.go`)
   - Central DI container managing all dependencies
   - Initializes components in correct order
   - Builds middleware chain

2. **Middleware Chain** (order matters!)
   - **Validation**: Request size, headers, JSON structure
   - **Auth**: API key validation and transformation
   - **Rate Limiter**: Per-client request/second limits with TTL
   - **Token Limiter**: Per-client token/minute limits
   - **Proxy**: HTTP reverse proxy to upstream

3. **Interfaces** (`internal/interfaces/interfaces.go`)
   - All major components implement interfaces
   - Enables testing with mocks
   - Future extensibility without breaking changes

### Critical Patterns

#### Concurrent Rate Limiting with TTL
```go
// Per-client isolation with automatic cleanup
type PerClientRateLimiterWithTTL struct {
    mu       sync.RWMutex  // Read-heavy optimization
    clients  map[string]*clientInfo
    ttl      time.Duration // Default: 1 hour
}

// Cleanup goroutine prevents memory leaks
go limiter.StartCleanup(5*time.Minute, stopChan)
```

#### Zero-Downtime Configuration
- File-based config with environment override
- `ConfigLoader` interface enables hot-reload (future)
- Graceful shutdown with 30-second timeout

#### Security Patterns
- API key masking in all logs
- Constant-time key comparison (timing attack prevention)
- Request validation against injection patterns

## 🧪 Mandatory TDD Workflow

### Test Organization

```
internal/
├── component/
│   ├── component.go          # Implementation
│   ├── component_test.go     # Unit tests (REQUIRED)
│   └── component_bench_test.go # Benchmarks (for performance-critical code)
tests/
├── integration_test.go       # Full request flow tests
└── e2e/                     # End-to-end test scenarios
    ├── auth_flow_test.go    # Complete auth scenarios
    ├── ratelimit_test.go    # Rate limiting under load
    └── proxy_chain_test.go  # Full middleware chain
```

### Test Requirements

- **Coverage**: Minimum 90%, critical paths 100%
- **Benchmarks**: Required for any code in hot paths
- **Race Detection**: All tests must pass with `-race`
- **Independence**: Tests must run in parallel
- **Integration**: Every feature must include end-to-end validation
- **Load Testing**: Rate limiters must handle 10K+ concurrent clients

### Using Test Agents

```bash
# Create comprehensive tests first
"Use nexus-test-designer to create tests for JWT authentication"

# Tests will include:
# - Unit tests for validation logic
# - Security tests for vulnerabilities  
# - Benchmarks for performance requirements
# - Integration tests for full flow
```

## 🛠️ Common Development Tasks

### Adding a New Feature

```bash
# 1. Create feature branch
git checkout -b feat/websocket-support

# 2. Start TDD with agents
"Implement WebSocket support for real-time updates"

# Agents automatically orchestrate:
# - Test design with WebSocket-specific patterns
# - Minimal implementation to pass tests
# - Performance optimization if needed
# - Code cleanup and documentation

# 3. Commit and push when complete
git add .
git commit -m "feat: add WebSocket support for real-time updates"
git push -u origin feat/websocket-support

# 4. Create PR
gh pr create
```

### Adding New Middleware

```bash
"Add request retry middleware with exponential backoff"

# nexus-test-designer will create tests for:
# - Retry logic with various response codes
# - Exponential backoff timing
# - Maximum retry limits
# - Integration with existing middleware
```

### Implementing New Rate Limiter

```bash
"Implement sliding window rate limiter"

# Agents ensure:
# - Interface compliance tests
# - Concurrency tests with race detection
# - Performance benchmarks < 1ms
# - TTL cleanup tests
```

## 📊 Key Patterns

### Interface Implementation

```go
// Always implement required interface
type YourRateLimiter struct {
    mu sync.RWMutex  // Use RWMutex for read-heavy operations
    // Implementation details
}

// Required methods
func (r *YourRateLimiter) Middleware(next http.Handler) http.Handler
func (r *YourRateLimiter) GetLimit(apiKey string) (allowed bool, remaining int)
func (r *YourRateLimiter) Reset(apiKey string)
```

### Middleware Pattern

```go
func YourMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Pre-processing
        
        // Call next or return early
        next.ServeHTTP(w, r)
        
        // Post-processing (if needed)
    })
}
```

### Error Handling

```go
// Always wrap errors with context
if err != nil {
    return fmt.Errorf("failed to validate key: %w", err)
}

// Security: Never leak sensitive info
logger.Error("Auth failed", map[string]any{
    "api_key": utils.MaskAPIKey(key),  // Always mask
    "error":   err.Error(),
})
```

## 🚀 Performance Targets

All code must meet these benchmarks:

- **Middleware Overhead**: < 1ms per layer
- **Rate Limit Check**: < 100μs
- **Token Counting**: < 500μs
- **Memory per Client**: < 1KB
- **Concurrent Clients**: 10,000+

## 🔒 Security Requirements

- **Input Validation**: All user input sanitized
- **API Keys**: Never logged in full, always masked
- **Timing Attacks**: Use constant-time comparison
- **Error Messages**: Never reveal system internals
- **Headers**: Sanitize against injection patterns

## 🧰 Development Commands

### Testing
```bash
make test                    # Run all tests
make test-coverage          # Generate coverage report
go test ./internal/proxy -v # Test specific package
go test -run TestName       # Run specific test
go test -bench=.           # Run benchmarks
go test -race              # Race detection
```

### Building
```bash
make build      # Build for current platform
make build-all  # Cross-platform builds
make docker     # Docker image
```

### Development
```bash
make dev        # Hot-reload server
make deps       # Update dependencies
golangci-lint run  # Lint code
```

## 🎯 Agent Quick Reference

| Task | Primary Agent | Automatic Flow |
|------|--------------|----------------|
| New Feature | nexus-test-designer | Tests → Implementation → Optimization → Refactor → Integration |
| Bug Fix | test-debugger | Debug → Fix → Test → Refactor |
| Performance | nexus-perf-optimizer | Profile → Optimize → Benchmark → Validate |
| Security | security-auditor | Audit → Test → Fix → Validate |
| Code Review | code-reviewer | Review → Feedback → Fix → Approve |
| Integration | nexus-integration-tester | Setup → Test Flows → Validate → Report |

## 📋 Configuration

- **Primary**: `config.yaml` in project root
- **Override**: `CONFIG_PATH` environment variable
- **Key Fields**:
  - `listen_port`: Gateway port (default: 8080)
  - `target_url`: Upstream API endpoint
  - `api_keys`: Client-to-upstream key mapping
  - `limits.requests_per_second`: Per-key rate limit
  - `limits.model_tokens_per_minute`: Per-key token limit

## 🔄 State Management

The project integrates with Cline's memory-bank system:
- Current state tracked in `memory-bank/state.md`
- Project index in `memory-bank/index.md`
- Update state after significant changes

## ⚡ Quick Start

```bash
# 1. Start development server
make dev

# 2. Create feature branch
git checkout -b feat/custom-auth-headers

# 3. Implement new feature (ALWAYS use agents)
"Use nexus-test-designer to create tests for custom auth header support"

# 4. Run tests continuously
reflex -r '\.go$' -- go test ./...

# 5. Run integration tests before commit
"Use nexus-integration-tester to validate custom auth headers through full request flow"

# 6. Check coverage before commit
make test-coverage

# 7. Commit with descriptive message
git add .
git commit -m "feat: add support for custom auth headers

- Implements X-Custom-Auth header validation
- Adds configuration for custom header names
- Includes comprehensive test coverage"

# 8. Push and create PR
git push -u origin feat/custom-auth-headers
gh pr create --title "Add custom auth header support" \
  --body "Adds configurable auth headers with full test coverage"
```

## 🚨 Important Rules

1. **ALWAYS USE FEATURE BRANCHES**: Never commit directly to main/master
2. **NO CODE WITHOUT TESTS**: Every line of production code must have a failing test first
3. **USE AGENTS**: Manual coding bypasses quality gates - always use agents
4. **SECURITY FIRST**: When in doubt, choose the more secure option
5. **PERFORMANCE MATTERS**: Benchmark everything in hot paths
6. **CLEAN COMMITS**: Each commit should have tests green
7. **PR BEFORE MERGE**: All code must be reviewed via pull request

## 📝 Commit Message Convention

Follow conventional commits format:
```
feat: add new feature
fix: fix a bug
docs: documentation changes
test: add or modify tests
refactor: code refactoring
perf: performance improvements
chore: maintenance tasks
```

Example:
```bash
git commit -m "feat: implement sliding window rate limiter

- Add sliding window algorithm with configurable window size
- Implement Redis backend for distributed limiting
- Include comprehensive unit and integration tests
- Benchmark shows < 100μs overhead per request"
```

Remember: The agent system is not optional. It enforces quality, security, and performance standards that manual development might miss.