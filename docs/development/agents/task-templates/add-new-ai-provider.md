# Task Template: Add New AI Provider Support

This is a multi-agent task that requires coordination between several agents.

## Phase 1: Proxy Network Agent

```markdown
You are the Proxy & Networking Agent for the Nexus API Gateway project.
[Include proxy-network agent instructions]

Task: Add support for [PROVIDER_NAME] AI API
- Add provider configuration to the proxy
- Implement request/response transformation if needed
- Handle provider-specific headers and authentication
- Add integration tests
```

## Phase 2: Rate Limiter Agent

```markdown
You are the Rate Limiting & Traffic Control Agent for the Nexus API Gateway project.
[Include rate-limiter agent instructions]

Previous work: Proxy agent added [PROVIDER_NAME] support
Task: Implement token counting for [PROVIDER_NAME] models
- Research [PROVIDER_NAME]'s token counting method
- Add model configurations
- Implement accurate token counting
- Add comprehensive tests
```

## Phase 3: Metrics Observer Agent

```markdown
You are the Metrics & Observability Agent for the Nexus API Gateway project.
[Include metrics-observer agent instructions]

Previous work: 
- Proxy agent added [PROVIDER_NAME] support
- Rate limiter agent added token counting

Task: Add provider-specific metrics
- Add provider label to existing metrics
- Track provider-specific errors
- Add model-specific latency tracking
- Update dashboards
```

## Phase 4: Test Quality Agent

```markdown
You are the Testing & Quality Agent for the Nexus API Gateway project.
[Include test-quality agent instructions]

Previous work: New [PROVIDER_NAME] provider has been added
Task: Ensure comprehensive test coverage
- Review all new code for test coverage
- Add integration tests for the full flow
- Add provider to CI/CD test matrix
- Update documentation
```

## Checklist
- [ ] Provider routing implemented
- [ ] Token counting accurate
- [ ] Metrics tracking enabled
- [ ] Tests comprehensive
- [ ] Documentation updated
- [ ] Example usage added