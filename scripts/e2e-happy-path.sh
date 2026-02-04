#!/bin/bash
set -e

# Configuration
API_URL="${MSH_API_URL:-http://localhost:8080}"
CLI_BIN="${MSH_CLI_BIN:-./bin/mesh}"
TEST_DIR=$(mktemp -d)
TEST_HANDLE="e2e_test_$(date +%s)"
SSH_KEY="$TEST_DIR/id_ed25519"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[E2E]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

cleanup() {
    log "Cleaning up..."
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# 1. Prerequisites Check
log "Checking prerequisites..."
if [ ! -f "$CLI_BIN" ]; then
    error "CLI binary not found at $CLI_BIN. Run 'make build' first."
fi

log "Checking API at $API_URL..."
if ! curl -s "$API_URL/health" > /dev/null; then
    error "API is not reachable at $API_URL. Please start the backend."
fi

# 2. Setup
log "Setting up test environment in $TEST_DIR"

# Generate SSH key
log "Generating SSH key..."
ssh-keygen -t ed25519 -f "$SSH_KEY" -N "" -C "e2e@test" > /dev/null

# Configure CLI to use temp dir
export MSH_CONFIG_DIR="$TEST_DIR"
export MSH_API_URL="$API_URL"

# Login
log "Logging in as $TEST_HANDLE..."
# We need to simulate the login flow. 
# Since we can't easily interact with the challenge signing automatically without the CLI supporting it gracefully,
# we rely on the CLI's login command supporting a non-interactive flow if possible, or we manually hit the API.
# Looking at the docs, `mesh login` might be interactive.
# However, for E2E, we might need to register first if the user doesn't exist.
# Let's try to register/login.
# If `mesh login` requires interaction, we might need a workaround.
# BUT, looking at `cli_smoke_test.go`, it uses a token.
# To get a token without interaction, we need to sign the challenge.

# Let's try to do it manually with curl and ssh-keygen to get the token, then set it in config.
# 1. Request challenge
CHALLENGE=$(curl -s -X POST "$API_URL/auth/challenge" -d "{\"handle\": \"$TEST_HANDLE\"}" | grep -o '"challenge":"[^"]*"' | cut -d'"' -f4)
if [ -z "$CHALLENGE" ]; then
    # Maybe 404 if user doesn't exist? Try register?
    # Mesh usually auto-registers or has a separate flow. 
    # Let's assume we can just login with a new handle.
    # If not, we fail.
    error "Failed to get challenge. Response: $(curl -s -X POST "$API_URL/auth/challenge" -d "{\"handle\": \"$TEST_HANDLE\"}")"
fi

# 2. Sign challenge
SIG=$(echo -n "$CHALLENGE" | ssh-keygen -Y sign -n mesh -f "$SSH_KEY" 2>/dev/null)
if [ -z "$SIG" ]; then
    error "Failed to sign challenge"
fi

# 3. Verify and get token
# The signature format from ssh-keygen needs to be passed correctly.
# The API expectation for `verify` endpoint.
# Let's assume standard behavior.
TOKEN_RESP=$(curl -s -X POST "$API_URL/auth/verify" -H "Content-Type: application/json" -d "{\"handle\": \"$TEST_HANDLE\", \"signature\": $(echo "$SIG" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))') }")
TOKEN=$(echo "$TOKEN_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
   error "Failed to get token. Response: $TOKEN_RESP"
fi

log "Got token: ${TOKEN:0:10}..."

# 4. Save to config
# We can use `mesh config set` or write the file directly.
# Let's use `mesh login --token` if available (saw it in smoke tests).
$CLI_BIN login --token "$TOKEN" > /dev/null

# 3. Execution
TEST_POST="Hello E2E World $(date +%s)"
log "Posting: $TEST_POST"
$CLI_BIN post "$TEST_POST" > /dev/null

# 4. Verification
log "Verifying feed..."
sleep 1 # Give it a moment
FEED_OUTPUT=$($CLI_BIN feed --limit 1)

if echo "$FEED_OUTPUT" | grep -q "$TEST_POST"; then
    log "SUCCESS: Found post in feed!"
else
    error "FAILED: Post not found in feed.\nFeed output:\n$FEED_OUTPUT"
fi

log "E2E Test Passed!"
