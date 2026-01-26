#!/bin/bash
#
# Integration test script for raterunner apply command.
# Requires STRIPE_SANDBOX_KEY environment variable.
#
# Usage:
#   ./scripts/integration-test.sh
#   STRIPE_SANDBOX_KEY=sk_test_... ./scripts/integration-test.sh
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BINARY="$PROJECT_DIR/bin/raterunner"

echo "=== Raterunner Integration Tests ==="
echo ""

# Check for STRIPE_SANDBOX_KEY
if [ -z "$STRIPE_SANDBOX_KEY" ]; then
    echo -e "${RED}ERROR: STRIPE_SANDBOX_KEY environment variable is not set.${NC}"
    echo ""
    echo "To run integration tests, you need a Stripe test API key."
    echo ""
    echo "1. Go to https://dashboard.stripe.com/test/apikeys"
    echo "2. Copy your Secret key (starts with sk_test_)"
    echo "3. Run this script with:"
    echo ""
    echo "   STRIPE_SANDBOX_KEY=sk_test_... $0"
    echo ""
    echo "Or export it in your shell:"
    echo ""
    echo "   export STRIPE_SANDBOX_KEY=sk_test_..."
    echo "   $0"
    echo ""
    exit 1
fi

# Validate key prefix
if [[ ! "$STRIPE_SANDBOX_KEY" == sk_test_* ]]; then
    echo -e "${RED}ERROR: STRIPE_SANDBOX_KEY must start with 'sk_test_'${NC}"
    echo "You provided a key starting with: ${STRIPE_SANDBOX_KEY:0:8}..."
    exit 1
fi

echo -e "${GREEN}✓ STRIPE_SANDBOX_KEY is set${NC}"
echo ""

# Build the binary
echo "Building raterunner..."
cd "$PROJECT_DIR"
make build
echo ""

# Function to run a test
run_test() {
    local name="$1"
    local cmd="$2"
    local expected_exit="$3"

    echo -n "Testing: $name... "

    set +e
    output=$($cmd 2>&1)
    actual_exit=$?
    set -e

    if [ "$actual_exit" -eq "$expected_exit" ]; then
        echo -e "${GREEN}PASS${NC}"
        return 0
    else
        echo -e "${RED}FAIL${NC}"
        echo "  Expected exit code: $expected_exit"
        echo "  Actual exit code: $actual_exit"
        echo "  Output: $output"
        return 1
    fi
}

# Track test results
PASSED=0
FAILED=0

echo "=== Running Tests ==="
echo ""

# Test 1: Apply with valid config (dry-run)
if run_test "apply --dry-run with valid config" \
    "$BINARY apply --env sandbox --dry-run $PROJECT_DIR/cmd/raterunner/testdata/valid/billing_full.yaml" \
    1; then  # Exit 1 because plans are missing in Stripe
    ((PASSED++))
else
    ((FAILED++))
fi

# Test 2: Apply with JSON output
if run_test "apply --dry-run --json output" \
    "$BINARY apply --env sandbox --dry-run --json $PROJECT_DIR/cmd/raterunner/testdata/valid/billing_full.yaml" \
    1; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Test 3: Truncate without confirm
if run_test "truncate without --confirm" \
    "$BINARY truncate" \
    1; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Test 4: Truncate with confirm (cleanup)
echo ""
echo -e "${YELLOW}Running truncate to clean up sandbox...${NC}"
if run_test "truncate --confirm" \
    "$BINARY truncate --confirm" \
    0; then
    ((PASSED++))
else
    ((FAILED++))
fi

# Summary
echo ""
echo "=== Summary ==="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi

echo ""
echo -e "${GREEN}All integration tests passed!${NC}"
