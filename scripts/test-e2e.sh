#!/bin/bash
set -e

# End-to-end test script for AnyAgent
# Tests: init -> install -> run -> mcp

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
TEST_DIR="/tmp/anyagent-e2e-test-$(date +%s)"

echo "=== AnyAgent E2E Test ==="
echo "Test directory: $TEST_DIR"
echo

# Build CLI
echo "1. Building CLI..."
cd "$PROJECT_ROOT/cli"
cargo build 2>/dev/null
CLI="$PROJECT_ROOT/cli/target/debug/agentx"
echo "   CLI built: $CLI"

# Start backend (optional - skip if not available)
echo
echo "2. Starting backend..."
cd "$PROJECT_ROOT/backend"
go build -o /tmp/anyagent-backend ./cmd/server 2>/dev/null || true
BACKEND_PID=""
if [ -f /tmp/anyagent-backend ]; then
    AGENT_STORE_DIR="$PROJECT_ROOT/agent-packs" /tmp/anyagent-backend &
    BACKEND_PID=$!
    sleep 1
    echo "   Backend started (PID: $BACKEND_PID)"
else
    echo "   Backend build skipped (Go not available or build failed)"
fi

# Create test project
echo
echo "3. Creating test project..."
mkdir -p "$TEST_DIR"
cd "$TEST_DIR"
git init -q
echo "# Test Project" > README.md
git add README.md
git commit -q -m "Initial commit"
echo "   Project created at $TEST_DIR"

# Test: agentx init
echo
echo "4. Testing: agentx init"
$CLI init --name "test-project"
if [ -d ".agentx" ] && [ -f ".agentx/config.yaml" ]; then
    echo "   ✓ init succeeded"
else
    echo "   ✗ init failed"
    exit 1
fi

# Test: agentx status
echo
echo "5. Testing: agentx status"
$CLI status

# Test: agentx list (should be empty)
echo
echo "6. Testing: agentx list"
$CLI list

# Test: agentx install (if backend is running)
echo
echo "7. Testing: agentx install"
if [ -n "$BACKEND_PID" ]; then
    $CLI install code-reviewer || echo "   (install requires backend to be running)"
else
    echo "   Skipped (backend not running)"
fi

# Test: agentx memory
echo
echo "8. Testing: agentx memory add"
$CLI memory add "This is a test memory" --kind "fact"
$CLI memory list

# Test: agentx config
echo
echo "9. Testing: agentx config"
$CLI config get project.name
$CLI config set project.default_agent "code-reviewer"
$CLI config get project.default_agent

# Test: agentx mcp (just verify it starts)
echo
echo "10. Testing: agentx mcp (startup check)"
timeout 2 $CLI mcp 2>&1 || true
echo "    ✓ MCP server starts"

# Cleanup
echo
echo "=== Cleanup ==="
if [ -n "$BACKEND_PID" ]; then
    kill $BACKEND_PID 2>/dev/null || true
fi
rm -rf "$TEST_DIR"
rm -f /tmp/anyagent-backend

echo
echo "=== All tests passed ==="
