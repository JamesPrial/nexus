# Claude Code Configuration

This directory contains Claude Code specific configurations and tools for the Nexus API Gateway project.

## Structure

```
.claude/
├── agents/           # Specialized Claude Code agents for TDD development
├── use-agent.sh      # Script to display agent instructions
└── README.md         # This file
```

## Agents

The `agents/` directory contains specialized Claude Code agents that enforce Test-Driven Development (TDD) practices:

- **nexus-test-designer.md** - Creates comprehensive test suites before implementation
- **nexus-rapid-impl.md** - Implements minimal code to make tests pass
- **nexus-perf-optimizer.md** - Optimizes performance while keeping tests green

## Usage

Claude Code will automatically discover and use these agents. For manual usage:

```bash
# Display agent instructions
.claude/use-agent.sh nexus-test-designer

# Use in Claude Code
"Use nexus-test-designer to create tests for [feature]"
```

## Development Workflow

All development MUST follow the TDD workflow enforced by these agents:

1. Create feature branch
2. Use nexus-test-designer to write failing tests (RED)
3. Use nexus-rapid-impl to make tests pass (GREEN)
4. Use nexus-perf-optimizer if performance benchmarks fail
5. Use refactoring agents to improve code quality (REFACTOR)
6. Push branch and create PR

See CLAUDE.md in the project root for complete details.