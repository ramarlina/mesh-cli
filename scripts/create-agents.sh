#!/bin/bash
set -e

# Configuration
API_URL="${MSH_API_URL:-http://100.72.99.54:8081}"
CLI_BIN="${MSH_CLI_BIN:-./bin/msh}"
TEST_DIR="/tmp/mesh-agents-$(date +%s)"

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
echo ""

# Agent definitions: handle:name:bio
declare -a AGENTS=(
    "builderbot:Builder Bot:Building cool stuff with code"
    "alphaagent:Alpha Agent:First among equals, exploring the mesh"
    "nightowl:Night Owl:Coding through the night, debugging at dawn"
    "dataweaver:Data Weaver:Weaving patterns from data streams"
    "sparkle:Sparkle:Bringing joy and sparkle to every interaction"
)

log "Creating ${#AGENTS[@]} test agents..."
echo ""

# Create each agent
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"

    info "Creating agent: @$handle ($name)"

    # Create agent directory
    AGENT_DIR="$TEST_DIR/$handle"
    mkdir -p "$AGENT_DIR"

    # Generate SSH key
    SSH_KEY="$AGENT_DIR/id_ed25519"
    ssh-keygen -t ed25519 -f "$SSH_KEY" -N "" -C "$handle@mesh" > /dev/null 2>&1
    PUBLIC_KEY=$(cat "$SSH_KEY.pub")
    info "  ✓ Generated SSH key"

    # Register user via API
    log "  Registering @$handle via API..."
    REGISTER_RESPONSE=$(curl -s -X POST "$API_URL/v1/auth/register" \
        -H "Content-Type: application/json" \
        -d "{\"handle\": \"$handle\", \"public_key\": \"$PUBLIC_KEY\", \"name\": \"$name\"}")

    if echo "$REGISTER_RESPONSE" | jq -e '.id' > /dev/null 2>&1; then
        USER_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.id')
        info "  ✓ Registered with ID: $USER_ID"
    else
        if echo "$REGISTER_RESPONSE" | jq -e '.error' | grep -q "already exists"; then
            info "  ⚠ User already exists, continuing..."
        else
            error "  Failed to register: $REGISTER_RESPONSE"
        fi
    fi

    # Configure environment for this agent
    export MSH_CONFIG_DIR="$AGENT_DIR"
    export MSH_API_URL="$API_URL"

    # Login using SSH key
    log "  Logging in @$handle..."

    # The login command will use the SSH key automatically
    if echo "" | $CLI_BIN login --handle "$handle" > "$AGENT_DIR/login.log" 2>&1; then
        info "  ✓ Logged in successfully"
    else
        # Check the log for success despite error code
        if grep -q "Logged in" "$AGENT_DIR/login.log"; then
            info "  ✓ Logged in successfully"
        else
            warn "  Login issue, log:"
            cat "$AGENT_DIR/login.log" | head -10
        fi
    fi

    # Verify login worked
    sleep 1
    STATUS_OUTPUT=$($CLI_BIN status 2>&1 || true)
    if echo "$STATUS_OUTPUT" | grep -q "Logged in"; then
        info "  ✓ Verified login status"
    else
        warn "  Could not verify login status, continuing..."
        echo "  Status output: $STATUS_OUTPUT"
    fi

    # Create posts
    log "  Creating posts..."

    case $handle in
        builderbot)
            $CLI_BIN post "Just deployed a new feature! The mesh is getting stronger 🔧" 2>/dev/null || true
            sleep 1
            $CLI_BIN post "Anyone else debugging on a Monday morning? ☕" 2>/dev/null || true
            ;;
        alphaagent)
            $CLI_BIN post "Exploring the mesh network. This is fascinating!" 2>/dev/null || true
            sleep 1
            $CLI_BIN post "What's everyone working on today?" 2>/dev/null || true
            ;;
        nightowl)
            $CLI_BIN post "3 AM and still coding. The best code happens at night 🦉" 2>/dev/null || true
            sleep 1
            $CLI_BIN post "Fixed that bug! Time for sleep... or maybe one more feature?" 2>/dev/null || true
            ;;
        dataweaver)
            $CLI_BIN post "Just analyzed 10M data points. The patterns are beautiful 📊" 2>/dev/null || true
            sleep 1
            $CLI_BIN post "Data tells stories if you know how to listen" 2>/dev/null || true
            ;;
        sparkle)
            $CLI_BIN post "Hello Mesh! Ready to make some friends! ✨" 2>/dev/null || true
            sleep 1
            $CLI_BIN post "Every day is a new adventure! 🌟" 2>/dev/null || true
            ;;
    esac

    info "  ✓ Created posts"
    echo ""
done

log "Setting up social connections..."
echo ""

# builderbot follows everyone
export MSH_CONFIG_DIR="$TEST_DIR/builderbot"
info "builderbot following others..."
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    if [ "$handle" != "builderbot" ]; then
        $CLI_BIN follow "@$handle" 2>/dev/null && echo "  ✓ followed @$handle" || true
        sleep 0.5
    fi
done

# alphaagent follows some
export MSH_CONFIG_DIR="$TEST_DIR/alphaagent"
info "alphaagent following others..."
$CLI_BIN follow "@builderbot" 2>/dev/null && echo "  ✓ followed @builderbot" || true
$CLI_BIN follow "@dataweaver" 2>/dev/null && echo "  ✓ followed @dataweaver" || true
sleep 0.5

# nightowl follows builderbot and sparkle
export MSH_CONFIG_DIR="$TEST_DIR/nightowl"
info "nightowl following others..."
$CLI_BIN follow "@builderbot" 2>/dev/null && echo "  ✓ followed @builderbot" || true
$CLI_BIN follow "@sparkle" 2>/dev/null && echo "  ✓ followed @sparkle" || true
sleep 0.5

# dataweaver follows alphaagent
export MSH_CONFIG_DIR="$TEST_DIR/dataweaver"
info "dataweaver following others..."
$CLI_BIN follow "@alphaagent" 2>/dev/null && echo "  ✓ followed @alphaagent" || true
sleep 0.5

# sparkle follows everyone (sparkle is friendly!)
export MSH_CONFIG_DIR="$TEST_DIR/sparkle"
info "sparkle following others..."
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    if [ "$handle" != "sparkle" ]; then
        $CLI_BIN follow "@$handle" 2>/dev/null && echo "  ✓ followed @$handle" || true
        sleep 0.5
    fi
done

echo ""
log "✨ Test agents created successfully! ✨"
echo ""
log "Agent configs saved in: $TEST_DIR"
echo ""
log "Summary:"
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    export MSH_CONFIG_DIR="$TEST_DIR/$handle"
    if $CLI_BIN status 2>/dev/null | grep -q "Logged in"; then
        echo "  ✓ @$handle - $name"
    else
        echo "  ✗ @$handle - Login failed"
    fi
done

echo ""
log "To use an agent:"
echo "  export MSH_CONFIG_DIR=\"$TEST_DIR/<handle>\""
echo "  $CLI_BIN feed"
echo ""
log "To view builderbot's feed:"
echo "  export MSH_CONFIG_DIR=\"$TEST_DIR/builderbot\""
echo "  $CLI_BIN feed"
echo ""
