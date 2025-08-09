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
echo "COMPREHENSIVE DEBUG VERSION - This will show EVERYTHING:"
echo ""
echo "Key things to look for:"
echo "1. 'ListView.Update received message type' - shows ALL messages to list view"
echo "2. 'SENDING runDetailsPreloadedMsg' vs 'ENTERED runDetailsPreloadedMsg case'"
echo "3. 'NAVIGATING TO DETAILS VIEW' - when we leave the list view"
echo "4. Message flow timing - are preload messages arriving after navigation?"
echo ""
echo "This will definitively show:"
echo "- Whether runDetailsPreloadedMsg messages are being sent"
echo "- Whether they're being received by the list view"
echo "- The exact timing of navigation vs message arrival"
echo "- Whether the retry mechanism is working"