# Nexus Architecture

## Overview
Nexus is a high-performance API gateway built with clean dependency injection architecture. It provides a secure, rate-limited proxy for AI model APIs with comprehensive monitoring capabilities.

## Request Flow

The gateway processes requests through a carefully ordered middleware chain:

```
Client Request
    ↓
HTTP Server (port 8080)
    ↓
Validation Middleware
    - Request size limits (10MB max)
    - Header validation
    - JSON structure validation
    ↓ (400/413 on failure)
Authentication Middleware  
    - API key extraction from Authorization header
    - Key validation against configured mappings
    - Upstream key transformation
    ↓ (401/403 on failure)
Rate Limiting Middleware
    - Per-client request/second limits
    - Burst capacity handling
    - TTL-based cleanup (1 hour)
    ↓ (429 on limit exceeded)
Token Limiting Middleware
    - AI model token usage tracking
    - Per-minute limits per client
    - Automatic window sliding
    ↓ (429 on limit exceeded)
Proxy Handler
    - HTTP reverse proxy to upstream API
    - Header forwarding and transformation
    - Response streaming
    ↓
Response to Client
```

## Key Components

### Container (Dependency Injection)
`internal/container/container.go`

The central DI container manages all component lifecycles and dependencies:
- Initializes components in correct order
- Builds middleware chain
- Manages configuration loading
- Handles graceful shutdown

### Gateway Service
`internal/gateway/service.go`

The main HTTP service that:
- Hosts the middleware chain
- Provides the `/health` endpoint
- Manages graceful shutdown with connection draining
- Coordinates metrics flushing on shutdown

### Middleware Components

#### Validation Middleware
`internal/middleware/validation.go`
- Enforces request size limits (10MB default)
- Validates required headers
- Protects against malformed requests

#### Authentication Middleware
`internal/auth/middleware.go`
- Extracts API keys from Authorization header
- Maps client keys to upstream keys
- Uses constant-time comparison for security
- Masks keys in all log output

#### Rate Limiter
`internal/proxy/ratelimiter_ttl.go`
- Per-client isolation with automatic TTL cleanup
- Configurable requests/second and burst limits
- Memory-efficient implementation using sync.RWMutex
- Background cleanup goroutine prevents memory leaks

#### Token Limiter
`internal/proxy/tokenlimiter.go`
- Tracks AI model token usage per client
- Sliding window implementation (per minute)
- Estimates tokens from request/response content
- Configurable limits per API key

### Metrics Collection
`internal/metrics/collector.go`

Collects and exports usage metrics:
- Request counts per client
- Token usage statistics
- Response times and status codes
- Configurable export formats (JSON/CSV)

**Known Issue:** The metrics collector may not initialize properly when enabled in configuration. This is tracked in [FUTURE_IMPROVEMENTS.md](FUTURE_IMPROVEMENTS.md).

## Configuration

The system uses file-based configuration with environment variable overrides:

```yaml
listen_port: 8080
target_url: "https://api.openai.com"
api_keys:
  "client-key": "upstream-key"
limits:
  requests_per_second: 10
  burst: 20
  model_tokens_per_minute: 100000
metrics:
  enabled: true
  export_path: "./metrics"
  export_interval: "5m"
```

## Health Monitoring

The `/health` endpoint provides:
- Service status confirmation
- Version information
- Timestamp for uptime tracking
- No authentication required for easy monitoring

Example response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "timestamp": "2025-01-03T12:00:00Z"
}
```

## Security Considerations

1. **API Key Security**
   - Keys never logged in full (always masked)
   - Constant-time comparison prevents timing attacks
   - Client keys isolated from upstream keys

2. **Input Validation**
   - Request size limits prevent DoS
   - Header injection protection
   - JSON structure validation

3. **Rate Limiting**
   - Per-client isolation prevents noisy neighbor issues
   - Automatic cleanup prevents memory exhaustion
   - Configurable limits per API key

## Performance Characteristics

- **Middleware Overhead:** < 1ms per layer
- **Rate Limit Check:** < 100μs
- **Token Counting:** < 500μs  
- **Memory per Client:** < 1KB
- **Concurrent Clients:** 10,000+ tested

## Testing Strategy

The codebase maintains 73% test coverage with focus on:
- Unit tests for all components
- Integration tests for full request flow
- Race condition detection
- Benchmark tests for performance-critical paths

## Future Enhancements

See [FUTURE_IMPROVEMENTS.md](FUTURE_IMPROVEMENTS.md) for planned enhancements including:
- OpenTelemetry integration
- Circuit breaker pattern
- Hot configuration reload
- WebSocket support