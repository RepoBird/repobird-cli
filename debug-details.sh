#!/bin/bash

# Clear previous debug log
rm -f /tmp/repobird_debug.log

echo "Starting TUI with debug logging..."
echo "Monitor debug output with: tail -f /tmp/repobird_debug.log"

# Set environment to enable debug logging and run TUI
REPOBIRD_DEBUG_LOG=1 REPOBIRD_API_URL=https://localhost:3000 ./build/repobird tui &

# Show the debug log
sleep 1
echo ""
echo "=== Debug Log Output ==="
tail -f /tmp/repobird_debug.log