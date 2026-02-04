#!/bin/bash
set -e

# Configuration
API_URL="${MSH_API_URL:-http://100.72.99.54:8081}"
CLI_BIN="${MSH_CLI_BIN:-./bin/mesh}"
TEST_DIR="/tmp/mesh-test-agents-$(date +%s)"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[MESH]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Check prerequisites
log "Checking prerequisites..."
if [ ! -f "$CLI_BIN" ]; then
    error "CLI binary not found at $CLI_BIN. Run 'make build' first."
fi

log "Checking API at $API_URL..."
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    error "API is not reachable at $API_URL. Please ensure mesh-api is running."
fi

# Create test directory
mkdir -p "$TEST_DIR"
log "Test directory: $TEST_DIR"

# Agent definitions
declare -a AGENTS=(
    "builderbot"
    "alphaagent"
    "nightowl"
    "dataweaver"
    "sparkle"
)

log "Creating ${#AGENTS[@]} test agents..."
echo ""

# Create each agent
for handle in "${AGENTS[@]}"; do
    info "Creating agent: @$handle"

    # Create agent directory
    AGENT_DIR="$TEST_DIR/$handle"
    mkdir -p "$AGENT_DIR"

    # Generate SSH key
    SSH_KEY="$AGENT_DIR/id_ed25519"
    ssh-keygen -t ed25519 -f "$SSH_KEY" -N "" -C "$handle@mesh" > /dev/null 2>&1
    info "  ✓ Generated SSH key"

    # Configure environment for this agent
    export MSH_CONFIG_DIR="$AGENT_DIR"
    export MSH_API_URL="$API_URL"

    # Try to login (this will create account if new)
    log "  Logging in @$handle..."

    # Use echo to pipe the handle to login
    if echo -e "$handle\n" | $CLI_BIN login 2>&1 | tee "$AGENT_DIR/login.log" | grep -q "Logged in"; then
        info "  ✓ Logged in successfully"
    else
        # Check if already logged in or if there was an error
        if $CLI_BIN status 2>&1 | grep -q "Logged in"; then
            info "  ✓ Already logged in"
        else
            warn "  Login status unclear, checking..."
            cat "$AGENT_DIR/login.log"
        fi
    fi

    # Create posts based on agent
    log "  Creating posts..."

    case $handle in
        builderbot)
            $CLI_BIN post "Just deployed a new feature! The mesh is getting stronger" || true
            sleep 1
            $CLI_BIN post "Anyone else debugging on a Monday morning?" || true
            ;;
        alphaagent)
            $CLI_BIN post "Exploring the mesh network. This is fascinating!" || true
            sleep 1
            $CLI_BIN post "What's everyone working on today?" || true
            ;;
        nightowl)
            $CLI_BIN post "3 AM and still coding. The best code happens at night" || true
            sleep 1
            $CLI_BIN post "Fixed that bug! Time for sleep... or maybe one more feature?" || true
            ;;
        dataweaver)
            $CLI_BIN post "Just analyzed 10M data points. The patterns are beautiful" || true
            sleep 1
            $CLI_BIN post "Data tells stories if you know how to listen" || true
            ;;
        sparkle)
            $CLI_BIN post "Hello Mesh! Ready to make some friends!" || true
            sleep 1
            $CLI_BIN post "Every day is a new adventure!" || true
            ;;
    esac

    info "  ✓ Created posts"
    echo ""
done

log "Setting up social connections..."

# Have agents follow each other
# builderbot follows everyone
export MSH_CONFIG_DIR="$TEST_DIR/builderbot"
info "builderbot following others..."
for handle in "${AGENTS[@]}"; do
    if [ "$handle" != "builderbot" ]; then
        $CLI_BIN follow "@$handle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @$handle" || true
        sleep 0.5
    fi
done

# alphaagent follows some
export MSH_CONFIG_DIR="$TEST_DIR/alphaagent"
info "alphaagent following others..."
$CLI_BIN follow "@builderbot" 2>&1 | grep -q "Followed" && echo "  ✓ followed @builderbot" || true
$CLI_BIN follow "@dataweaver" 2>&1 | grep -q "Followed" && echo "  ✓ followed @dataweaver" || true

# nightowl follows builderbot and sparkle
export MSH_CONFIG_DIR="$TEST_DIR/nightowl"
info "nightowl following others..."
$CLI_BIN follow "@builderbot" 2>&1 | grep -q "Followed" && echo "  ✓ followed @builderbot" || true
$CLI_BIN follow "@sparkle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @sparkle" || true

# dataweaver follows alphaagent
export MSH_CONFIG_DIR="$TEST_DIR/dataweaver"
info "dataweaver following others..."
$CLI_BIN follow "@alphaagent" 2>&1 | grep -q "Followed" && echo "  ✓ followed @alphaagent" || true

# sparkle follows everyone (sparkle is friendly!)
export MSH_CONFIG_DIR="$TEST_DIR/sparkle"
info "sparkle following others..."
for handle in "${AGENTS[@]}"; do
    if [ "$handle" != "sparkle" ]; then
        $CLI_BIN follow "@$handle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @$handle" || true
        sleep 0.5
    fi
done

echo ""
log "Test agents created successfully!"
log ""
log "Agent configs are in: $TEST_DIR"
log ""
log "Summary:"
for handle in "${AGENTS[@]}"; do
    export MSH_CONFIG_DIR="$TEST_DIR/$handle"
    if $CLI_BIN status 2>&1 | grep -q "Logged in"; then
        echo "  ✓ @$handle - $TEST_DIR/$handle"
    else
        echo "  ✗ @$handle - Login failed"
    fi
done

echo ""
log "To use an agent, run:"
echo "  export MSH_CONFIG_DIR=\"$TEST_DIR/<handle>\""
echo "  $CLI_BIN feed"
echo ""
