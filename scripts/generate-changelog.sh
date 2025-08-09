#!/bin/bash
set -e

# Generate changelog from git commits
# Usage: ./scripts/generate-changelog.sh [range] [format]
# Examples:
#   ./scripts/generate-changelog.sh                    # All commits
#   ./scripts/generate-changelog.sh v1.0.0..HEAD      # Since v1.0.0
#   ./scripts/generate-changelog.sh HEAD~10..HEAD     # Last 10 commits
#   ./scripts/generate-changelog.sh v1.0.0..HEAD md   # Markdown format

RANGE="${1:-}"
FORMAT="${2:-text}"

# Function to format commit for different outputs
format_commit() {
    local hash=$1
    local subject=$2
    local body=$3
    local author=$4
    local date=$5
    
    case $FORMAT in
        markdown|md)
            echo "- $subject ([${hash:0:7}](https://github.com/repobird/repobird-cli/commit/$hash))"
            ;;
        json)
            echo "  {"
            echo "    \"hash\": \"$hash\","
            echo "    \"subject\": \"$subject\","
            echo "    \"author\": \"$author\","
            echo "    \"date\": \"$date\""
            echo "  }"
            ;;
        text|*)
            echo "* $subject ($hash)"
            ;;
    esac
}

# Function to categorize commits
categorize_commit() {
    local subject=$1
    
    case $subject in
        feat*|feature*) echo "Features" ;;
        fix*|bugfix*) echo "Bug Fixes" ;;
        docs*|doc*) echo "Documentation" ;;
        style*) echo "Style Changes" ;;
        refactor*) echo "Code Refactoring" ;;
        perf*|performance*) echo "Performance Improvements" ;;
        test*) echo "Tests" ;;
        build*|ci*|chore*) echo "Build System & CI" ;;
        security*) echo "Security" ;;
        breaking*|BREAKING*) echo "Breaking Changes" ;;
        *) echo "Other Changes" ;;
    esac
}

main() {
    echo "Generating changelog..."
    
    if [ "$FORMAT" = "json" ]; then
        echo "{"
        echo "  \"changelog\": ["
    elif [ "$FORMAT" = "markdown" ] || [ "$FORMAT" = "md" ]; then
        echo "# Changelog"
        echo ""
    fi
    
    # Get git log with custom format
    local git_range=""
    if [ -n "$RANGE" ]; then
        git_range="$RANGE"
    fi
    
    # Associative array to group commits by category
    declare -A categories
    
    while IFS=$'\t' read -r hash subject body author date; do
        # Skip merge commits
        if [[ $subject == Merge* ]]; then
            continue
        fi
        
        # Categorize commit
        category=$(categorize_commit "$subject")
        
        # Format commit
        formatted=$(format_commit "$hash" "$subject" "$body" "$author" "$date")
        
        # Add to category
        if [ -z "${categories[$category]}" ]; then
            categories[$category]="$formatted"
        else
            categories[$category]="${categories[$category]}"$'\n'"$formatted"
        fi
        
    done < <(git log $git_range --pretty=format:"%H%x09%s%x09%b%x09%an%x09%ad" --date=short --reverse)
    
    # Output categorized commits
    local first=true
    for category in "Breaking Changes" "Features" "Bug Fixes" "Security" "Performance Improvements" "Documentation" "Code Refactoring" "Tests" "Build System & CI" "Style Changes" "Other Changes"; do
        if [ -n "${categories[$category]}" ]; then
            if [ "$FORMAT" = "json" ]; then
                if [ "$first" = false ]; then
                    echo ","
                fi
                echo "    {"
                echo "      \"category\": \"$category\","
                echo "      \"commits\": ["
                echo "${categories[$category]}"
                echo "      ]"
                echo "    }"
                first=false
            elif [ "$FORMAT" = "markdown" ] || [ "$FORMAT" = "md" ]; then
                echo "## $category"
                echo ""
                echo "${categories[$category]}"
                echo ""
            else
                echo "$category:"
                echo "${categories[$category]}"
                echo ""
            fi
        fi
    done
    
    if [ "$FORMAT" = "json" ]; then
        echo "  ]"
        echo "}"
    fi
}

main "$@"