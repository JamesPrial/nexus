# Nexus Agent Orchestration Guide

This guide explains how to use the enhanced agent system for the Nexus API Gateway project.

## Agent Directory Structure

```
nexus/
├── agents/                      # Agent definitions
│   ├── nexus-test-designer.md   # TDD test creation
│   ├── nexus-rapid-impl.md      # Minimal implementation
│   ├── nexus-perf-optimizer.md  # Performance optimization
│   └── ...                      # More specialized agents
├── .claude/
│   └── agents/                  # Symlink to agents/ for auto-discovery
```

## Setting Up Agents

```bash
# Create symlink for Claude Code to discover agents
cd nexus
mkdir -p .claude
ln -s ../agents .claude/agents

# Verify agents are available
ls .claude/agents/
```

## Agent Workflows

### 1. Standard TDD Feature Development

```bash
# Start with requirement
"Implement JWT authentication for the API gateway"

# Workflow:
# 1. nexus-test-designer creates comprehensive tests
# 2. nexus-rapid-impl makes tests pass
# 3. code-refactor improves implementation
# 4. test-refactor optimizes tests
```

### 2. Performance-Critical Feature

```bash
# Requirement with performance SLA
"Add request deduplication with < 100μs overhead"

# Workflow:
# 1. nexus-test-designer includes performance benchmarks
# 2. nexus-rapid-impl creates working version
# 3. nexus-perf-optimizer optimizes for benchmarks
# 4. code-refactor cleans up optimized code
```

### 3. Security Feature

```bash
# Security-sensitive requirement
"Implement API key rotation without downtime"

# Workflow:
# 1. nexus-test-designer includes security tests
# 2. nexus-rapid-impl basic implementation
# 3. security-auditor reviews for vulnerabilities
# 4. nexus-rapid-impl fixes security issues
# 5. code-refactor improves structure
```

## Automatic Agent Triggers

### File-Based Triggers

When working in Claude Code, agents are triggered by file patterns:

- Creating `*_test.go` → Activates implementation agents
- Modifying `internal/auth/*` → Activates auth specialist
- Benchmark failures → Activates performance optimizer
- Test failures → Activates test debugger

### Context-Based Triggers

Agents understand development context:

```bash
# After writing tests (RED phase)
"All tests are failing as expected"
# → Automatically suggests nexus-rapid-impl

# After implementation (GREEN phase)  
"All tests passing but code is messy"
# → Automatically suggests code-refactor

# When benchmarks fail
"BenchmarkRateLimit: 2ms per op (exceeds 1ms limit)"
# → Automatically activates nexus-perf-optimizer
```

## Multi-Agent Orchestration Examples

### Example 1: Complete Rate Limiter Feature

```bash
"Implement sliding window rate limiter with Redis backend"

# Agent flow:
nexus-test-designer:
  - Creates unit tests for sliding window algorithm
  - Creates integration tests for Redis
  - Creates benchmarks for performance
  
nexus-rapid-impl:
  - Implements basic sliding window
  - Adds Redis connection
  - Makes all tests pass

nexus-perf-optimizer:
  - Optimizes Redis operations
  - Adds connection pooling
  - Reduces allocations

code-refactor:
  - Extracts interfaces
  - Improves error handling
  - Adds documentation
```

### Example 2: API Gateway Security Hardening

```bash
"Harden the gateway against common attacks"

# Agent flow:
security-auditor:
  - Identifies vulnerabilities
  - Creates security test suite

nexus-test-designer:
  - Converts security requirements to tests
  - Adds fuzzing tests
  - Creates penetration test scenarios

nexus-rapid-impl:
  - Implements security fixes
  - Adds input validation
  - Implements rate limiting

code-reviewer:
  - Reviews security implementation
  - Checks for new vulnerabilities
```

## Agent Communication

### Context Handoff Format

Agents pass structured context:

```yaml
handoff:
  from: nexus-test-designer
  to: nexus-rapid-impl
  tdd_phase: red->green
  
  context:
    tests_created:
      - internal/auth/jwt_test.go (15 tests)
      - internal/auth/jwt_bench_test.go (3 benchmarks)
    
    requirements:
      - JWT validation < 100μs
      - Support RS256 and HS256
      - Graceful expiry handling
    
    run_command: go test ./internal/auth -v
    
    critical_tests:
      - TestJWT_ValidToken_Success
      - TestJWT_ExpiredToken_Rejected
      - BenchmarkJWT_Validation
```

### Long-Running Context

For large features, context is preserved:

```yaml
project_context:
  feature: advanced_rate_limiting
  phase: implementation
  
  decisions:
    - Using Redis for distributed state
    - Sliding window algorithm chosen
    - 1-second window granularity
  
  completed:
    - Basic rate limiter tests
    - Redis integration tests
    
  in_progress:
    - Performance optimization
    
  blockers:
    - Redis connection pooling issues
```

## Best Practices

### 1. Let Agents Drive Development

```bash
# DON'T: Write code first
"I implemented rate limiting, can you add tests?"

# DO: Start with tests
"Use nexus-test-designer to create rate limiting tests"
```

### 2. Trust Agent Expertise

```bash
# DON'T: Override agent recommendations
"Ignore the performance suggestions and use my approach"

# DO: Work with agent expertise
"Why does nexus-perf-optimizer recommend this approach?"
```

### 3. Use Appropriate Models

- **Haiku agents**: Quick tasks, simple implementations
- **Sonnet agents**: Standard development, testing
- **Opus agents**: Complex optimization, architecture

### 4. Maintain TDD Discipline

```bash
# Always follow RED -> GREEN -> REFACTOR
1. "Create tests for feature X" (RED)
2. "Make tests pass" (GREEN)  
3. "Improve code quality" (REFACTOR)
```

## Troubleshooting

### Agent Not Triggering

```bash
# Be explicit about agent needs
"Use nexus-test-designer to create comprehensive tests for the new auth feature"

# Provide context
"The auth tests are failing. Use test-debugger to investigate"
```

### Context Lost Between Agents

```bash
# Request context preservation
"Use context-manager to summarize progress before switching agents"

# Reference previous work
"Continue from where nexus-rapid-impl left off"
```

### Performance Goals Not Met

```bash
# Provide specific targets
"nexus-perf-optimizer: achieve < 100μs latency for auth checks"

# Share profiling data
"Here's the CPU profile showing the bottleneck"
```

## Advanced Orchestration

### Parallel Agent Execution

For independent tasks:

```bash
"In parallel:
1. nexus-test-designer: Create metrics collection tests
2. api-documenter: Document the metrics endpoints
3. terraform-specialist: Prepare infrastructure for metrics storage"
```

### Conditional Flows

Based on results:

```bash
"Run benchmarks. If performance < 1ms, proceed to refactoring.
Otherwise, use nexus-perf-optimizer first."
```

### Review Loops

For critical features:

```bash
"Implement auth feature with review loop:
1. nexus-test-designer → tests
2. nexus-rapid-impl → implementation
3. security-auditor → review
4. If issues found, go to step 2
5. code-refactor → final cleanup"
```

## Integration with Development Workflow

### 1. Feature Branch

```bash
git checkout -b feat/advanced-rate-limiting

# Start TDD cycle
"Use nexus-test-designer for rate limiting feature"
```

### 2. Continuous Testing

```bash
# Watch mode during development
reflex -r '\.go$' -- go test ./...

# Agents react to test results
```

### 3. PR Review

```bash
# Before creating PR
"Use code-reviewer to check all changes"
"Use test-coverage-analyst to verify coverage"
```

### 4. Performance Validation

```bash
# Before merge
"Run all benchmarks and verify SLAs are met"
"Use nexus-perf-optimizer if any regressions found"
```

Remember: The agent system is designed to enhance your development workflow, not replace your judgment. Use agents as expert advisors while maintaining overall project vision.