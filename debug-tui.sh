#!/bin/bash

# Clear debug log
echo "Clearing debug log..."
> /tmp/repobird_debug.log

echo "Debug log cleared. You can now test the TUI and check /tmp/repobird_debug.log for debugging information."
echo ""
echo "To view debug log in real-time while testing:"
echo "tail -f /tmp/repobird_debug.log"
echo ""
echo "To run the TUI:"
echo "make tui"