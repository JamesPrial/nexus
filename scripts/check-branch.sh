#!/bin/bash
# Branch check script to ensure feature branch workflow

BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null)

# Check if we're in a git repository
if [ $? -ne 0 ]; then
    echo "❌ ERROR: Not in a git repository"
    exit 1
fi

# Check if on main/master
if [[ "$BRANCH" == "main" || "$BRANCH" == "master" ]]; then
    echo "❌ ERROR: You are on $BRANCH branch!"
    echo ""
    echo "YOU MUST CREATE A FEATURE BRANCH FIRST:"
    echo "  git checkout -b feat/your-feature-name"
    echo ""
    echo "This is MANDATORY per the development workflow."
    echo "See PREFLIGHT-CHECKLIST.md for details."
    echo ""
    echo "Current branch: $BRANCH"
    exit 1
fi

# Success
echo "✅ On feature branch: $BRANCH"
exit 0