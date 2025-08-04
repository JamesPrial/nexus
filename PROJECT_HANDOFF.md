# Nexus Project Handoff Documentation

## Project Overview
Nexus is a self-hosted API gateway for AI model APIs (like OpenAI) written in Go. It provides:
- Rate limiting (requests/second)
- Token limiting (tokens/minute)
- API key management (client → upstream mapping)
- Metrics collection
- Clean dependency injection architecture

## Current State (After Phase 1)
- ✅ All tests passing
- ✅ Critical linting issues resolved
- ✅ PR #16 merged fixing test stability
- ⚠️ Some architectural issues remain

## Critical Context

### 1. Development Workflow (MANDATORY)
```bash
# NEVER work on main branch
git checkout -b feat/[description]

# Run tests before ANY commit
make test

# Run linter before pushing
/tmp/golangci-lint run --timeout=5m  # or install your own

# Create detailed PRs
gh pr create --title "feat: description" --body "..."
```

### 2. Architecture Issues Found

#### Import Path Inconsistency
- **Problem**: Mixed imports `github.com/jamesprial/nexus/config` vs `internal/config`
- **Location**: `internal/container/container.go`
- **Fix**: Standardize to `internal/*` throughout

#### Incomplete Branch Consolidation
- **Problem**: PR #15 merged multiple branches causing confusion
- **Evidence**: Commit messages mention "2 remaining issues" that were fixed in Phase 1

#### Agent Workflow Complexity
- **Problem**: Over-engineered TDD workflow with 7+ specialized agents
- **Location**: `/home/jamesprial/claude/nexus/agents/` directory
- **Consider**: Simplifying or better documenting the workflow

## Phase 2: Clean Up Technical Debt

### Priority Tasks
1. **Fix Import Paths**
   ```go
   // Change from:
   import "github.com/jamesprial/nexus/config"
   // To:
   import "github.com/jamesprial/nexus/internal/config"
   ```
   Files: Check all `*.go` files, especially in `internal/container/`

2. **Remove Dead Code**
   - Check for unused functions after branch consolidation
   - Use `go mod tidy` to clean dependencies
   - Remove any duplicate test utilities

3. **Simplify Agent Workflow**
   - Review agents in `agents/` directory
   - Document which ones are actually needed
   - Consider consolidating overlapping agents

4. **Update Documentation**
   - Update README.md with actual project state
   - Fix any outdated references in CLAUDE.md
   - Document the real development workflow (not the idealized one)

## Phase 3: Feature Improvements

### Recommended Implementations
1. **Health Check Endpoint**
   ```go
   // Add to internal/gateway/service.go
   router.HandleFunc("/health", healthHandler)
   ```

2. **Graceful Shutdown**
   - Already has 30-second timeout in main.go
   - Add connection draining
   - Add metrics flush on shutdown

3. **Integration Tests**
   - Current coverage: 61.4% total
   - Focus on `internal/gateway` (only 10.7% coverage)
   - Add end-to-end tests for full request flow

4. **OpenTelemetry Support**
   - Add tracing middleware
   - Export metrics in OTLP format
   - Keep existing Prometheus support

## Phase 4: Project Organization

### Tasks
1. **Separate overnight-tools**
   ```bash
   cd /home/jamesprial/claude
   git init overnight-tools
   mv overnight-tools/* overnight-tools/.git/
   git add . && git commit -m "Initial commit"
   gh repo create overnight-tools --private
   ```

2. **GitHub Releases**
   - Use existing Makefile targets
   - Tag with semantic versioning
   - Include binaries from `make build-all`

3. **CI/CD Pipeline**
   ```yaml
   # .github/workflows/release.yml
   - Run tests
   - Run linter  
   - Build binaries
   - Create release
   ```

## Key Files to Understand

### Core Architecture
- `internal/interfaces/interfaces.go` - All component interfaces
- `internal/container/container.go` - Dependency injection setup
- `cmd/gateway/main.go` - Entry point and server setup

### Middleware Chain (order matters!)
1. `internal/middleware/validation.go` - Request validation
2. `internal/auth/middleware.go` - API key auth
3. `internal/proxy/ratelimiter.go` - Rate limiting
4. `internal/proxy/tokenlimiter.go` - Token limiting
5. `internal/proxy/proxy.go` - Reverse proxy

### Configuration
- `config.yaml` - Main config file
- `internal/config/loader.go` - Config loading logic

## Testing Guidelines

### Required Checks
```bash
# All must pass before PR
make test                    # Unit tests
go test -race ./...         # Race detection
/tmp/golangci-lint run      # Linting
go test -cover ./...        # Coverage check
```

### Test Organization
```
internal/[package]/
├── [package].go           # Implementation
├── [package]_test.go      # Unit tests
└── integration_test.go    # Integration tests (if needed)
```

## Common Pitfalls

1. **Never commit to main** - Git hooks will block you
2. **Always run linter** - Don't create "fix linting" PRs
3. **Test isolation** - Each test must create fresh state
4. **API key masking** - Never log full API keys

## Quick Command Reference

```bash
# Development
make dev                    # Start with hot reload
make test                   # Run all tests
make test-coverage         # Generate coverage report

# Building
make build                 # Build for current platform
make build-all            # Cross-platform builds
make docker               # Build Docker image

# Utilities
go mod tidy               # Clean up dependencies
git checkout -b feat/name # Create feature branch
gh pr create              # Create pull request
```

## Contact & Resources

- GitHub Repo: https://github.com/JamesPrial/nexus
- PR #16: Test stabilization (reference for code style)
- Original design: API gateway with clean architecture

Remember: The codebase is over-documented but under-tested. Trust the code more than the docs.