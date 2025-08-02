# Agent Usage Examples

## Example 1: Using the Rate Limiter Agent

### Task: "Add support for Anthropic Claude model token counting"

**Instructions to provide:**
```markdown
You are the Rate Limiting & Traffic Control Agent for the Nexus API Gateway project.
[Include content from docs/agents/rate-limiter-agent.md]

Task: Add support for token counting for Anthropic Claude models (claude-3-opus, claude-3-sonnet, claude-3-haiku)
```

**Expected Agent Behavior:**
1. Research Anthropic's token counting method
2. Implement token counting in `/internal/proxy/tokencounter.go`
3. Add tests for Claude model token counting
4. Update documentation

---

## Example 2: Using the Auth Security Agent

### Task: "Implement API key rotation mechanism"

**Instructions to provide:**
```markdown
You are the Authentication & Security Agent for the Nexus API Gateway project.
[Include content from docs/agents/auth-security-agent.md]

Task: Implement API key rotation that allows old and new keys to work during transition period
```

**Expected Agent Behavior:**
1. Design rotation mechanism with grace period
2. Modify `/internal/auth/file_key_manager.go`
3. Add rotation commands to CLI
4. Ensure zero-downtime rotation
5. Add comprehensive security tests

---

## Example 3: Cross-Agent Collaboration

### Task: "Add usage-based billing metrics"

**Step 1 - Metrics Agent:**
```markdown
You are the Metrics & Observability Agent.
Task: Implement usage tracking metrics for billing (requests, tokens, cost estimation)
```

**Step 2 - Rate Limiter Agent:**
```markdown
You are the Rate Limiting & Traffic Control Agent.
Previous work: Metrics agent added usage tracking
Task: Add usage-based rate limiting that cuts off at billing limits
```

**Step 3 - Auth Agent:**
```markdown
You are the Authentication & Security Agent.
Previous work: Metrics tracks usage, rate limiter enforces limits
Task: Add billing tier information to API key configuration
```

---

## Example 4: Using Agent for Code Review

### Task: "Review the new metrics implementation"

**Instructions to provide:**
```markdown
You are the Testing & Quality Agent for the Nexus API Gateway project.
[Include test-quality agent instructions]

Task: Review the metrics implementation in /internal/metrics/ for:
- Test coverage
- Performance impact
- Integration with existing systems
- Best practices
```

---

## Practical Workflow

### 1. Single Agent Task
```bash
# 1. Identify the agent needed
# 2. Copy the agent instructions from docs/agents/[agent-name].md
# 3. Provide the instructions + specific task to Claude Code
# 4. Agent works within its domain expertise
```

### 2. Multi-Agent Task
```bash
# 1. Break down the task by agent domains
# 2. Start with the primary agent
# 3. Use handoff protocol to move between agents
# 4. Each agent maintains its specialized focus
```

### 3. Agent Context in Practice

When working on rate limiting:
```markdown
Context: You are the rate-limiter agent
Your expertise: Rate limiting algorithms, performance optimization
Your domain: /internal/proxy/ratelimiter*.go
Focus on: Performance < 1ms overhead, support 10k+ clients
```

When working on security:
```markdown
Context: You are the auth-security agent  
Your expertise: Security best practices, credential management
Your domain: /internal/auth/
Focus on: Never compromise security, no credential leaks
```

---

## Tips for Effective Agent Usage

1. **Be Specific**: Provide the exact agent role and domain
2. **Include Context**: Share relevant previous work from other agents
3. **Set Boundaries**: Remind agents of their domain limits
4. **Use Handoffs**: Explicitly hand off between agents for complex tasks
5. **Maintain Focus**: Don't let agents drift outside their expertise

## Example Handoff

```markdown
FROM: rate-limiter agent
TO: metrics-observer agent

I've implemented the new sliding window rate limiter in:
- /internal/proxy/ratelimiter_sliding.go
- Added tests in ratelimiter_sliding_test.go

Please add metrics for:
- Sliding window hit/miss rates
- Memory usage per window
- Processing time histogram

The rate limiter exposes a Stats() method you can use.
```