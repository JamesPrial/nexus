# Phase 2: Technical Debt Cleanup

## Quick Start
```bash
cd /home/jamesprial/claude/nexus
git checkout main && git pull
git checkout -b fix/technical-debt
```

## Priority 1: Fix Import Paths
```bash
# Find all incorrect imports
grep -r "github.com/jamesprial/nexus/config" --include="*.go" .

# Should use internal paths instead:
# BAD:  import "github.com/jamesprial/nexus/config"  
# GOOD: import "github.com/jamesprial/nexus/internal/config"
```

Key file: `internal/container/container.go` (has mixed imports)

## Priority 2: Remove Dead Code
```bash
# Find potentially unused code
go mod tidy
golangci-lint run --enable=unused,deadcode

# Check for duplicate test helpers after branch merge
find . -name "*_test.go" -exec grep -l "mock\|fake\|stub" {} \;
```

## Priority 3: Simplify Agent Workflow
The `agents/` directory has 7+ TDD enforcement agents. Most are unused.
- Keep: `nexus-test-designer`, `nexus-rapid-impl` 
- Remove or document: The rest
- Update: CLAUDE.md to reflect reality

## Priority 4: Fix Documentation
1. README.md - Remove aspirational features, document what exists
2. CLAUDE.md - Simplify the workflow section
3. Add: ARCHITECTURE.md with real system design

## Checklist
- [ ] All imports use internal/ paths
- [ ] No unused dependencies in go.mod
- [ ] Agent workflow documented or removed
- [ ] Documentation matches reality
- [ ] All tests still pass
- [ ] Linter passes

## Success Criteria
- Cleaner, more maintainable codebase
- New developers can understand project in 10 minutes
- No "fix: linting" PRs needed later