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
echo "- 'runDetailsPreloadedMsg received' (are messages being handled?)"
echo "- 'Successfully cached run' (is caching working?)"
echo "- 'Run X is still preloading, adding small delay' (timing fix)"
echo "- 'Retry successful - using cached data' (retry mechanism working)"
echo ""
echo "The fix adds a small delay if a run is still preloading when Enter is pressed."
echo "This should resolve the timing issue where messages arrive after navigation."