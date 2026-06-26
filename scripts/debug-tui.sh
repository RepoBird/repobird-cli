#!/bin/bash

# Get the script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"

# Create logs directory if it doesn't exist
mkdir -p "$PROJECT_DIR/logs"

# Clear debug log in new location
echo "Clearing debug logs..."
> "$PROJECT_DIR/logs/repobird_debug.log"

# Also clear old log location for completeness
> /tmp/repobird_debug.log 2>/dev/null || true

echo "Debug logs cleared in $PROJECT_DIR/logs/"
echo ""
echo "Starting TUI with debug logging enabled..."
echo "Logs will be written to: $PROJECT_DIR/logs/repobird_debug.log"
echo ""

# Build and run the TUI
cd "$PROJECT_DIR"

# Load .env file if it exists
if [ -f "$PROJECT_DIR/.env" ]; then
    echo "Loading environment from .env file..."
    set -a  # Mark all new variables for export
    source "$PROJECT_DIR/.env"
    set +a  # Stop marking for export
    echo "  Loaded REPOBIRD_API_URL: ${REPOBIRD_API_URL:-not set}"
fi

if ! make build; then
    echo "Build failed. Cannot run TUI."
    exit 1
fi
REPOBIRD_ENV=dev REPOBIRD_DEBUG_LOG=1 ./build/repobird tui