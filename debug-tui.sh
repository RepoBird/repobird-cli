#!/bin/bash

# Clear debug log
echo "Clearing debug log..."
> /tmp/repobird_debug.log

echo "Debug log cleared. Enhanced debugging added to track preloading."
echo ""
echo "Now test again:"
echo "1. Run: make tui"
echo "2. Wait for runs to load"
echo "3. Scroll down and press Enter on a run"
echo "4. Check logs: cat /tmp/repobird_debug.log"
echo ""
echo "Look for these debug messages:"
echo "- 'preloadRunDetails called'"  
echo "- 'Will preload X runs'"
echo "- 'Starting API call'"
echo "- 'Cached run with key'"
echo ""
echo "This will show us exactly why preloading isn't working."