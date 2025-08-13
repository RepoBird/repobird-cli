#!/bin/bash

# Script to run tests and show a summary
# Usage: ./scripts/test-summary.sh [test command and args]

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Default test command if none provided
if [ $# -eq 0 ]; then
    TEST_CMD="go test -v -race -timeout 30s ./..."
else
    TEST_CMD="$@"
fi

# Run tests and capture output
echo "Running tests..."
START_TIME=$(date +%s)
OUTPUT=$(REPOBIRD_ENV=dev $TEST_CMD 2>&1)
TEST_EXIT_CODE=$?
END_TIME=$(date +%s)
TOTAL_TIME=$((END_TIME - START_TIME))

# Only show failures and build errors, not all the passing tests
echo "$OUTPUT" | grep -E "(^FAIL|^---\s+FAIL:|^\s+[a-zA-Z_][a-zA-Z0-9_]*\.go:[0-9]+:|panic:|Error:|build failed)"

# Extract counts
TOTAL_TESTS=$(echo "$OUTPUT" | grep -E "^---\s+(PASS|FAIL|SKIP)" | wc -l)
PASSED=$(echo "$OUTPUT" | grep -E "^---\s+PASS:" | wc -l)
FAILED=$(echo "$OUTPUT" | grep -E "^---\s+FAIL:" | wc -l)
SKIPPED=$(echo "$OUTPUT" | grep -E "^---\s+SKIP:" | wc -l)

# Get failed tests with the most useful info available
FAILED_TESTS_INFO=$(echo "$OUTPUT" | grep -E "^---\s+FAIL:" | sed 's/^---\s*FAIL:\s*//' | sed 's/ (.*)$//' | while read test_name; do
    # Look for file reference in the output near this test
    file_ref=$(echo "$OUTPUT" | grep -A 10 -B 5 "$test_name" | grep -E "^\s+[a-zA-Z_][a-zA-Z0-9_]*\.go:[0-9]+" | head -1 | awk '{print $1}' | sed 's/:.*//')
    if [ -n "$file_ref" ]; then
        echo "$file_ref ($test_name)"
    else
        echo "$test_name"
    fi
done | sort -u)

# Calculate test passing rate
if [ $TOTAL_TESTS -gt 0 ]; then
    PASS_RATE=$(echo "scale=1; $PASSED * 100 / $TOTAL_TESTS" | bc 2>/dev/null || echo "0")
else
    PASS_RATE=0
fi

# Format time display
if [ $TOTAL_TIME -ge 60 ]; then
    MINUTES=$((TOTAL_TIME / 60))
    SECONDS=$((TOTAL_TIME % 60))
    TIME_DISPLAY="${MINUTES}m ${SECONDS}s"
else
    TIME_DISPLAY="${TOTAL_TIME}s"
fi

# Print compact summary
echo ""
echo -e "${BOLD}========================================"

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}✅ ALL TESTS PASSED!${NC}"
    echo -e "${BOLD}========================================${NC}"
    echo -e "${GREEN}Passed: $PASSED/$TOTAL_TESTS (${PASS_RATE}%)${NC}"
    if [ $SKIPPED -gt 0 ]; then
        echo -e "${YELLOW}Skipped: $SKIPPED${NC}"
    fi
    echo -e "${BLUE}Time: ${TIME_DISPLAY}${NC}"
else
    echo -e "${RED}❌ TESTS FAILED${NC}"
    echo -e "${BOLD}========================================${NC}"
    
    # Show failed tests if any
    if [ $FAILED -gt 0 ]; then
        echo -e "${RED}Failed Tests:${NC}"
        while IFS= read -r test_info; do
            if [ -n "$test_info" ]; then
                echo -e "  ${RED}•${NC} $test_info"
            fi
        done <<< "$FAILED_TESTS_INFO"
        echo ""
    fi
    
    # Show test counts on one line
    echo -e "Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}, ${YELLOW}$SKIPPED skipped${NC} / $TOTAL_TESTS total (${PASS_RATE}%) ${BLUE}(${TIME_DISPLAY})${NC}"
fi

echo -e "${BOLD}========================================${NC}"

# Exit with the same code as the tests
exit $TEST_EXIT_CODE