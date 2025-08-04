# Nexus Project Handoff Index

Start here when picking up this project.

## 📋 Documents Overview

1. **[PROJECT_STATUS.md](PROJECT_STATUS.md)** - Quick 2-minute overview of everything
2. **[PROJECT_HANDOFF.md](PROJECT_HANDOFF.md)** - Comprehensive context and background

## 🔧 Phase-Specific Guides

- **[PHASE_2_TECH_DEBT.md](PHASE_2_TECH_DEBT.md)** - Clean up technical debt (2-3 hours)
- **[PHASE_3_FEATURES.md](PHASE_3_FEATURES.md)** - Add features & improve tests (3-4 hours)  
- **[PHASE_4_ORGANIZATION.md](PHASE_4_ORGANIZATION.md)** - Organize & release (1-2 hours)

## 🚀 Quick Start

```bash
# 1. Check project state
cd /home/jamesprial/claude/nexus
git status  # Should be on main with PR #16 merged
make test   # Should pass

# 2. Pick your phase and follow its guide
git checkout -b feat/[phase-name]

# 3. Essential commands
make test                  # Run tests
/tmp/golangci-lint run    # Check linting
make dev                  # Start dev server
```

## ⚠️ Critical Rules

1. **NEVER commit to main** - Always use feature branches
2. **Test before commit** - `make test` must pass
3. **Lint before PR** - No "fix linting" commits
4. **Document reality** - Not aspirations

Good luck! The project is stable after Phase 1, just needs cleanup and polish.