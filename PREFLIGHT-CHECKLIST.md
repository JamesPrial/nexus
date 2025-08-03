# 🚨 PREFLIGHT CHECKLIST FOR CLAUDE CODE 🚨

**STOP! Before writing ANY code, you MUST complete this checklist:**

## ✅ MANDATORY CHECKS (IN ORDER)

### 1. ⚠️ FEATURE BRANCH CHECK
```bash
git branch --show-current
```
- [ ] ❌ If output is `main` or `master` → **STOP! Create feature branch first!**
- [ ] ✅ If output is `feat/*` or similar → Continue

**TO FIX:** 
```bash
git checkout -b feat/[feature-name]
```

### 2. 📋 WORKING DIRECTORY CHECK
```bash
pwd
```
- [ ] Confirm you're in the correct project directory

### 3. 🔄 SYNC CHECK
```bash
git pull origin main
```
- [ ] Ensure your branch is up to date with main

### 4. 🧪 TEST STATUS CHECK
```bash
go test ./...
```
- [ ] All existing tests pass before starting work

## ⛔ CRITICAL RULES

1. **NEVER SKIP THE BRANCH CHECK** - Even for "small" changes
2. **NEVER WORK ON MAIN/MASTER** - No exceptions
3. **ALWAYS USE AGENTS** - nexus-test-designer → nexus-rapid-impl → code-refactor
4. **ALWAYS UPDATE MEMORY-BANK** - After file modifications

## 🔴 RED FLAGS

If you find yourself:
- Working without creating a feature branch first
- Committing directly to main/master
- Implementing before writing tests
- Skipping the agent workflow

**STOP IMMEDIATELY** and restart following this checklist.

## 🎯 CORRECT WORKFLOW

1. `git checkout -b feat/[name]` - ALWAYS FIRST
2. Use nexus-test-designer - Create failing tests
3. Use nexus-rapid-impl - Make tests pass
4. Use code-refactor - Clean up code
5. Update memory-bank/state.md
6. Commit with clear message
7. Push branch and create PR

## 🚨 ENFORCEMENT

This checklist is enforced by:
- Git pre-commit hooks (prevent main commits)
- Agent pre-checks (verify branch before running)
- Memory-bank validation (check branch in state updates)

**Remember:** The process exists to ensure quality, enable collaboration, and prevent mistakes. Following it is NOT OPTIONAL.