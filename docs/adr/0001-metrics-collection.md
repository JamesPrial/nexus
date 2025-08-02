# ADR 0001: Metrics Collection Implementation

## Context

Nexus is a self-hosted API gateway for rate limiting and proxying AI APIs. To enhance observability, we need to implement aggregated metrics collection focusing on cost control (token usage), security (per-key tracking), and performance (latency, error rates). This should be lightweight, avoiding per-request logging, and emphasize aggregates.

Key requirements:
- Aggregated counters and histograms per API key, endpoint, and model.
- Follow RED method (Rate, Errors, Duration).
- Minimal dependencies, in-memory storage initially.
- Expose via a protected /metrics endpoint.

## Decision

- **Metrics Type**: Aggregated counters (total requests, successes, failures, tokens) and histograms (latency percentiles) per key, endpoint, and model.
- **Collection Method**: HTTP middleware to record post-request, using sync.Mutex for thread-safety and atomic counters for performance.
- **Exposure**: JSON via /metrics endpoint; optionally Prometheus format using github.com/prometheus/client_golang/prometheus.
- **Security**: Authenticate access to /metrics.
- **Performance**: Non-blocking updates.
- **Extensibility**: Interface-based for future backends (e.g., Redis).
- **Implementation**: New internal/metrics package with collector, middleware, and exporter.

Add dependency: github.com/prometheus/client_golang/prometheus for histograms and export.

## Status

Proposed

## Consequences

- Improves monitoring without heavy overhead.
- Adds a dependency, but it's standard for Go metrics.
- Enables future extensions like persistence or advanced monitoring.
- Requires updates to config, container, and gateway for integration.

## Alternatives Considered

- Fully custom in-memory without Prometheus: Simpler but less standard for histograms.
- External systems like StatsD: Too heavy for lightweight gateway.
- Database-backed: Adds complexity and persistence needs, deferred for now.
