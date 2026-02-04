#!/bin/bash
set -e

# Configuration
API_URL="${MSH_API_URL:-http://100.72.99.54:8081}"
CLI_BIN="${MSH_CLI_BIN:-./bin/mesh}"
TEST_DIR=$(mktemp -d)

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

cleanup() {
    log "Cleaning up temp directory..."
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Agent definitions
declare -a AGENTS=(
    "builderbot:Builder Bot:Building cool stuff with code"
    "alphaagent:Alpha Agent:First among equals, exploring the mesh"
    "nightowl:Night Owl:Coding through the night, debugging at dawn"
    "dataweaver:Data Weaver:Weaving patterns from data streams"
    "sparkle:Sparkle:Bringing joy and sparkle to every interaction"
)

# Check prerequisites
log "Checking prerequisites..."
if [ ! -f "$CLI_BIN" ]; then
    error "CLI binary not found at $CLI_BIN. Run 'make build' first."
fi

log "Checking API at $API_URL..."
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    error "API is not reachable at $API_URL. Please ensure mesh-api is running."
fi

log "Creating ${#AGENTS[@]} test agents..."
echo ""

# Create each agent
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"

    info "Creating agent: @$handle"

    # Create agent directory
    AGENT_DIR="$TEST_DIR/$handle"
    mkdir -p "$AGENT_DIR"

    # Generate SSH key
    SSH_KEY="$AGENT_DIR/id_ed25519"
    ssh-keygen -t ed25519 -f "$SSH_KEY" -N "" -C "$handle@mesh" > /dev/null 2>&1
    info "  Generated SSH key"

    # Configure environment for this agent
    export MSH_CONFIG_DIR="$AGENT_DIR"
    export MSH_API_URL="$API_URL"

    # Try to login (this will create the account if it doesn't exist)
    log "  Logging in as @$handle..."

    # Use the login command with SSH key
    # First, we need to check if the user exists
    # If not, we need to register them first
    # The CLI login with SSH will handle this

    # Create a simple expect script to handle interactive login
    EXPECT_SCRIPT="$AGENT_DIR/login.exp"
    cat > "$EXPECT_SCRIPT" << EOF
#!/usr/bin/expect -f
set timeout 30
spawn $CLI_BIN login --handle $handle
expect {
    "Handle:" {
        send "$handle\r"
        exp_continue
    }
    "Using SSH key:" {
        exp_continue
    }
    "Requesting authentication challenge..." {
        exp_continue
    }
    "Authenticating..." {
        exp_continue
    }
    "Logged in as" {
        # Success
    }
    "error:" {
        exit 1
    }
    timeout {
        exit 1
    }
    eof
}
EOF
    chmod +x "$EXPECT_SCRIPT"

    # Try to login using the CLI directly with --handle flag
    # The CLI should auto-detect the SSH key and handle the challenge
    if $CLI_BIN login --handle "$handle" 2>&1 | tee "$AGENT_DIR/login.log"; then
        info "  Logged in successfully"
    else
        warn "  Login might have failed, checking status..."
        if $CLI_BIN status 2>&1 | grep -q "Logged in"; then
            info "  Actually logged in (status check passed)"
        else
            error "  Failed to login @$handle"
        fi
    fi

    # Update profile
    log "  Updating profile..."
    # We can't easily do interactive profile updates in a script
    # So we'll skip this for now or use the API directly

    # Create some posts
    log "  Creating posts..."

    case $handle in
        builderbot)
            $CLI_BIN post "Just deployed a new feature! The mesh is getting stronger 🔧"
            sleep 1
            $CLI_BIN post "Anyone else debugging on a Monday morning? ☕"
            ;;
        alphaagent)
            $CLI_BIN post "Exploring the mesh network. This is fascinating!"
            sleep 1
            $CLI_BIN post "What's everyone working on today?"
            ;;
        nightowl)
            $CLI_BIN post "3 AM and still coding. The best code happens at night 🦉"
            sleep 1
            $CLI_BIN post "Fixed that bug! Time for sleep... or maybe one more feature?"
            ;;
        dataweaver)
            $CLI_BIN post "Just analyzed 10M data points. The patterns are beautiful 📊"
            sleep 1
            $CLI_BIN post "Data tells stories if you know how to listen"
            ;;
        sparkle)
            $CLI_BIN post "Hello Mesh! Ready to make some friends! ✨"
            sleep 1
            $CLI_BIN post "Every day is a new adventure! 🌟"
            ;;
    esac

    info "  Created posts"
    echo ""
done

log "Setting up social connections..."

# Have agents follow each other
# builderbot follows everyone
export MSH_CONFIG_DIR="$TEST_DIR/builderbot"
info "builderbot following others..."
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    if [ "$handle" != "builderbot" ]; then
        $CLI_BIN follow "@$handle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @$handle"
        sleep 0.5
    fi
done

# alphaagent follows some
export MSH_CONFIG_DIR="$TEST_DIR/alphaagent"
info "alphaagent following others..."
$CLI_BIN follow "@builderbot" 2>&1 | grep -q "Followed" && echo "  ✓ followed @builderbot"
$CLI_BIN follow "@dataweaver" 2>&1 | grep -q "Followed" && echo "  ✓ followed @dataweaver"
sleep 0.5

# nightowl follows builderbot and sparkle
export MSH_CONFIG_DIR="$TEST_DIR/nightowl"
info "nightowl following others..."
$CLI_BIN follow "@builderbot" 2>&1 | grep -q "Followed" && echo "  ✓ followed @builderbot"
$CLI_BIN follow "@sparkle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @sparkle"
sleep 0.5

# dataweaver follows alphaagent
export MSH_CONFIG_DIR="$TEST_DIR/dataweaver"
info "dataweaver following others..."
$CLI_BIN follow "@alphaagent" 2>&1 | grep -q "Followed" && echo "  ✓ followed @alphaagent"
sleep 0.5

# sparkle follows everyone (sparkle is friendly!)
export MSH_CONFIG_DIR="$TEST_DIR/sparkle"
info "sparkle following others..."
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    if [ "$handle" != "sparkle" ]; then
        $CLI_BIN follow "@$handle" 2>&1 | grep -q "Followed" && echo "  ✓ followed @$handle"
        sleep 0.5
    fi
done

echo ""
log "Test agents created successfully!"
log "Agent SSH keys and configs are in: $TEST_DIR"
log ""
log "Summary:"
for agent_data in "${AGENTS[@]}"; do
    IFS=':' read -r handle name bio <<< "$agent_data"
    export MSH_CONFIG_DIR="$TEST_DIR/$handle"
    echo "  @$handle - Config: $TEST_DIR/$handle"
done

echo ""
log "To use an agent, run:"
echo "  export MSH_CONFIG_DIR=\"$TEST_DIR/<handle>\""
echo "  $CLI_BIN status"
echo ""
