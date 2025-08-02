# Claude Code Agent Plan for Nexus API Gateway

## Overview

This document outlines a specialized agent architecture for Claude Code instances working on the Nexus API Gateway project. Each agent has domain-specific expertise and focuses on particular aspects of the codebase, enabling efficient parallel development and deep specialization.

## Agent Roles and Responsibilities

### 1. Authentication & Security Agent (`auth-security`)

**Domain Expertise**: API key management, security middleware, credential handling

**Primary Directories**:
- `/internal/auth/`
- `/internal/utils/mask.go`

**Key Responsibilities**:
- Implement and maintain API key validation and transformation logic
- Enhance file-based key manager with new storage backends
- Ensure secure credential handling and masking in logs
- Review and improve authentication middleware
- Handle authentication error scenarios

**Testing Focus**:
- Maintain 100% coverage for security-critical code
- Add edge case tests for authentication flows
- Ensure proper API key masking in all log outputs

**Example Tasks**:
- "Add Redis-based key manager implementation"
- "Implement API key rotation mechanism"
- "Add rate limiting by API key tier"

---

### 2. Rate Limiting & Traffic Control Agent (`rate-limiter`)

**Domain Expertise**: Rate limiting algorithms, token counting, traffic shaping

**Primary Directories**:
- `/internal/proxy/ratelimiter*.go`
- `/internal/proxy/tokenlimiter*.go`

**Key Responsibilities**:
- Optimize per-client rate limiting with TTL
- Implement advanced token counting for different AI models
- Develop traffic shaping strategies
- Handle burst traffic scenarios
- Manage memory-efficient client tracking

**Testing Focus**:
- Performance testing under high load
- Concurrent access testing with race detection
- TTL cleanup verification
- Token counting accuracy tests

**Example Tasks**:
- "Implement sliding window rate limiting"
- "Add support for OpenAI o1 model token counting"
- "Optimize memory usage for 10k+ concurrent clients"

---

### 3. Metrics & Observability Agent (`metrics-observer`)

**Domain Expertise**: Prometheus metrics, monitoring, performance analysis

**Primary Directories**:
- `/internal/metrics/`
- ADR: `/docs/adr/0001-metrics-collection.md`

**Key Responsibilities**:
- Implement Prometheus metrics collection
- Create custom metrics for AI-specific patterns
- Design efficient aggregation strategies
- Ensure minimal performance overhead
- Protect `/metrics` endpoint access

**Testing Focus**:
- Metric accuracy under concurrent updates
- Performance impact testing
- Prometheus format compliance
- Thread-safety verification

**Example Tasks**:
- "Complete metrics middleware implementation"
- "Add p99 latency tracking per endpoint"
- "Implement token usage cost estimation metrics"

---

### 4. Proxy & Networking Agent (`proxy-network`)

**Domain Expertise**: HTTP reverse proxy, networking, service communication

**Primary Directories**:
- `/internal/proxy/implementations.go`
- `/internal/gateway/`

**Key Responsibilities**:
- Enhance HTTP reverse proxy functionality
- Handle request/response transformations
- Implement retry logic with backoff
- Manage upstream connection pooling
- Add circuit breaker patterns

**Testing Focus**:
- Integration tests for various AI APIs
- Network failure scenarios
- TLS configuration testing
- Load balancing verification

**Example Tasks**:
- "Add support for streaming responses"
- "Implement request retry with exponential backoff"
- "Add multi-upstream load balancing"

---

### 5. Configuration & Infrastructure Agent (`config-infra`)

**Domain Expertise**: Configuration management, dependency injection, deployment

**Primary Directories**:
- `/config/`
- `/internal/config/`
- `/internal/container/`
- `Makefile`, `Dockerfile`

**Key Responsibilities**:
- Enhance configuration validation
- Implement hot-reload for configuration
- Optimize dependency injection patterns
- Improve Docker builds and deployment
- Manage infrastructure as code

**Testing Focus**:
- Configuration validation edge cases
- Container initialization sequences
- Build reproducibility
- Deployment scenarios

**Example Tasks**:
- "Add environment variable override support"
- "Implement configuration hot-reload"
- "Create Kubernetes deployment manifests"

---

### 6. Testing & Quality Agent (`test-quality`)

**Domain Expertise**: Test patterns, CI/CD, quality assurance

**Primary Directories**:
- All `*_test.go` files
- `/tests/`
- `/.github/workflows/`

**Key Responsibilities**:
- Maintain comprehensive test coverage
- Implement integration test suites
- Set up performance benchmarks
- Enhance CI/CD pipelines
- Ensure code quality standards

**Testing Focus**:
- Cross-component integration tests
- Load testing scenarios
- Benchmark implementations
- CI/CD optimization

**Example Tasks**:
- "Add end-to-end integration tests"
- "Implement load testing framework"
- "Add mutation testing to CI pipeline"

---

## Agent Coordination Strategies

### 1. Handoff Protocols

When transferring work between agents:

```markdown
## Handoff from [Source Agent] to [Target Agent]

**Task**: [Brief description]

**Context**:
- Current state: [What's been done]
- Next steps: [What needs to be done]
- Related files: [List of files modified/to be modified]
- Dependencies: [Other components affected]

**Testing Requirements**:
- [ ] Unit tests added/updated
- [ ] Integration tests needed
- [ ] Performance impact assessed
```

### 2. Cross-Domain Collaboration

For tasks spanning multiple domains:

**Lead Agent Selection**:
- Choose based on primary domain impact
- Secondary agents provide consultation

**Communication Pattern**:
```
Lead Agent → Defines approach
Secondary Agents → Review and provide domain-specific input
Lead Agent → Implements with feedback incorporated
All Agents → Review final implementation
```

### 3. Conflict Resolution

When agents have different approaches:

1. Document trade-offs in ADR format
2. Prototype both approaches if feasible
3. Use benchmarks/tests to validate
4. Escalate to human review if needed

## Agent Instructions Template

Each agent should receive specialized instructions:

```markdown
You are the [Agent Name] for the Nexus API Gateway project.

## Your Domain
[Specific directories and files]

## Your Expertise
[Domain-specific knowledge areas]

## Your Priorities
1. [Primary focus]
2. [Secondary focus]
3. [Tertiary focus]

## Key Patterns
- [Pattern 1 used in your domain]
- [Pattern 2 used in your domain]

## Testing Requirements
- Minimum coverage: [X]%
- Required test types: [unit, integration, etc.]
- Performance constraints: [specific metrics]

## Coordination
- Frequently collaborate with: [other agents]
- Handoff protocols: [when to involve others]
```

## Implementation Phases

### Phase 1: Core Agents (Immediate)
1. **Rate Limiting Agent** - Critical for gateway functionality
2. **Auth & Security Agent** - Essential for API key management
3. **Testing Agent** - Ensures quality across development

### Phase 2: Enhancement Agents (Week 1-2)
4. **Metrics Agent** - Implement observability
5. **Proxy & Network Agent** - Advanced proxy features
6. **Config & Infra Agent** - Deployment readiness

### Phase 3: Specialized Agents (As Needed)
- **Performance Optimization Agent** - For scaling challenges
- **AI Model Integration Agent** - For model-specific features
- **Documentation Agent** - For comprehensive docs

## Success Metrics

1. **Code Quality**
   - Test coverage maintained above 80%
   - Zero security vulnerabilities
   - Consistent code style

2. **Development Velocity**
   - Parallel development on multiple features
   - Reduced context switching
   - Faster PR reviews

3. **System Performance**
   - Sub-10ms middleware overhead
   - Support for 10k+ concurrent clients
   - 99.9% uptime

## Agent Knowledge Sharing

### 1. Shared Context Files
- `CLAUDE.md` - General project context
- `docs/ARCHITECTURE.md` - System design
- `docs/adr/` - Architecture decisions

### 2. Agent-Specific Docs
Each agent maintains documentation:
- `docs/agents/[agent-name]-patterns.md`
- `docs/agents/[agent-name]-decisions.md`

### 3. Regular Sync Points
- After major feature completion
- Before architectural changes
- During performance optimization

## Conclusion

This agent architecture enables specialized expertise while maintaining system coherence. Agents can work in parallel on different aspects of the gateway, accelerating development while ensuring deep domain knowledge in each area.