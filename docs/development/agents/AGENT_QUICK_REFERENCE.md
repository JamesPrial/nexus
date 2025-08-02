# Claude Code Agent Quick Reference

## Agent Directory

| Agent | Domain | Primary Focus | Key Files |
|-------|---------|--------------|-----------|
| `auth-security` | Authentication & Security | API keys, auth middleware | `/internal/auth/`, `/internal/utils/mask.go` |
| `rate-limiter` | Rate Limiting & Traffic | Request/token limits, traffic shaping | `/internal/proxy/ratelimiter*.go`, `tokenlimiter*.go` |
| `metrics-observer` | Metrics & Observability | Prometheus metrics, monitoring | `/internal/metrics/` |
| `proxy-network` | Proxy & Networking | HTTP proxy, request routing | `/internal/proxy/implementations.go`, `/internal/gateway/` |
| `config-infra` | Configuration & Infrastructure | Config, DI, deployment | `/config/`, `/internal/container/`, `Dockerfile` |
| `test-quality` | Testing & Quality | Test coverage, CI/CD | `*_test.go`, `/.github/workflows/` |

## Quick Agent Selection Guide

### "I need to work on..."

**API Key Management** → `auth-security`
- Adding new auth methods
- Key rotation
- Security headers
- Credential storage

**Rate Limiting** → `rate-limiter`
- New limiting algorithms
- Token counting
- Performance optimization
- Burst handling

**Monitoring/Metrics** → `metrics-observer`
- Adding new metrics
- Prometheus integration
- Performance tracking
- Cost monitoring

**HTTP Proxy Features** → `proxy-network`
- Request routing
- Retry logic
- Circuit breakers
- Load balancing

**Configuration** → `config-infra`
- New config options
- Environment variables
- Docker/K8s deployment
- Dependency injection

**Testing** → `test-quality`
- Test coverage
- Integration tests
- CI/CD pipeline
- Benchmarks

## Cross-Agent Tasks

### Feature: "Add support for new AI provider"
1. **Primary**: `proxy-network` - Add routing logic
2. **Secondary**: `rate-limiter` - Add token counting
3. **Secondary**: `auth-security` - Update key validation
4. **Secondary**: `metrics-observer` - Add provider metrics

### Feature: "Implement usage-based billing"
1. **Primary**: `metrics-observer` - Track usage metrics
2. **Secondary**: `rate-limiter` - Enforce usage limits
3. **Secondary**: `auth-security` - Add tier management

### Feature: "Add Redis support"
1. **Primary**: `config-infra` - Add Redis configuration
2. **Secondary**: `auth-security` - Redis key storage
3. **Secondary**: `rate-limiter` - Redis-backed limits

## Agent Communication Template

```markdown
TO: [Target Agent]
FROM: [Source Agent]
RE: [Feature/Task Name]

CONTEXT:
- What I've implemented: [brief description]
- What you need to do: [specific tasks]
- Key files I've modified: [list]
- Interfaces/contracts: [any new interfaces]

HANDOFF CHECKLIST:
- [ ] Tests passing
- [ ] Documentation updated
- [ ] No security concerns
- [ ] Performance validated
```

## Common Patterns Across Agents

### 1. Interface-Driven Design
All agents should respect and implement defined interfaces in `/internal/interfaces/`

### 2. Middleware Chain
Order matters: Validation → Auth → RateLimit → TokenLimit → Metrics → Proxy

### 3. Context Propagation
Use request context for passing API key, request ID, and other metadata

### 4. Error Handling
- Don't leak sensitive info in errors
- Use structured errors with codes
- Log errors appropriately

### 5. Testing Strategy
- Unit tests for components
- Integration tests for workflows
- Benchmarks for performance-critical code

## Performance Targets

All agents should maintain:
- Middleware overhead: < 1ms per layer
- Memory usage: O(1) per request
- Startup time: < 5 seconds
- Graceful shutdown: < 30 seconds

## Security Requirements

All agents must:
- Never log sensitive data
- Validate all inputs
- Fail securely (deny by default)
- Handle errors without info leakage
- Follow OWASP API Security Top 10

## Quick Commands

```bash
# Run specific agent's tests
go test ./internal/auth/...      # auth-security
go test ./internal/proxy/...     # rate-limiter & proxy-network  
go test ./internal/metrics/...   # metrics-observer

# Check coverage for agent domain
go test -cover ./internal/auth/...

# Run benchmarks
go test -bench=. ./internal/proxy/...

# Build and test everything
make test
```