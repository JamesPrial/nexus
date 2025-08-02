#!/bin/bash
# Script to display agent instructions for easy copying

AGENT=$1
AGENT_DIR="docs/agents"

if [ -z "$AGENT" ]; then
    echo "Usage: ./scripts/use-agent.sh <agent-name>"
    echo ""
    echo "Available agents:"
    echo "  - auth-security"
    echo "  - rate-limiter"
    echo "  - metrics-observer"
    echo "  - proxy-network"
    echo "  - config-infra"
    echo "  - test-quality"
    exit 1
fi

AGENT_FILE="$AGENT_DIR/$AGENT-agent.md"

if [ ! -f "$AGENT_FILE" ]; then
    echo "Error: Agent file not found: $AGENT_FILE"
    echo "Please check the agent name and try again."
    exit 1
fi

echo "=== COPY THE FOLLOWING TO USE THE $AGENT AGENT ==="
echo ""
cat "$AGENT_FILE"
echo ""
echo "=== END OF AGENT INSTRUCTIONS ==="
echo ""
echo "Now add your specific task after these instructions."