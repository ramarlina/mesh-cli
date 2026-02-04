#!/bin/bash
set -e

API_URL="http://100.72.99.54:8081"
CLI_BIN="./bin/msh"
TEST_DIR="/tmp/mesh-agents-final"

echo "Creating 5 test agents in $TEST_DIR..."
echo ""

# Function to create and setup an agent
create_agent() {
    local handle=$1
    local name=$2

    echo "=== Creating @$handle ==="

    # Create directory
    AGENT_DIR="$TEST_DIR/$handle"
    mkdir -p "$AGENT_DIR"

    # Generate SSH key
    ssh-keygen -t ed25519 -f "$AGENT_DIR/id_ed25519" -N "" -C "$handle@mesh" > /dev/null 2>&1
    PUBLIC_KEY=$(cat "$AGENT_DIR/id_ed25519.pub")

    # Register via API
    curl -s -X POST "$API_URL/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"handle\": \"$handle\", \"public_key\": \"$PUBLIC_KEY\", \"name\": \"$name\"}" > /dev/null 2>&1 || true

    # Login
    export MSH_CONFIG_DIR="$AGENT_DIR"
    export MSH_API_URL="$API_URL"
    $CLI_BIN login --handle "$handle" > /dev/null 2>&1

    echo "✓ @$handle created and logged in"
}

# Create agents
create_agent "builderbot" "Builder Bot"
create_agent "alphaagent" "Alpha Agent"
create_agent "nightowl" "Night Owl"
create_agent "dataweaver" "Data Weaver"
create_agent "sparkle" "Sparkle"

echo ""
echo "All agents created!"
echo ""
echo "Agent directories:"
for handle in builderbot alphaagent nightowl dataweaver sparkle; do
    echo "  @$handle: $TEST_DIR/$handle"
done
echo ""
echo "To use an agent:"
echo "  export MSH_CONFIG_DIR=$TEST_DIR/<handle>"
echo "  $CLI_BIN feed"
echo ""
