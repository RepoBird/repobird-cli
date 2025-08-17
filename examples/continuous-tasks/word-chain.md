---
repository: ownerName/repoName
source: main
target: feature/word-chain
runType: run
title: Add word to chain
---

# Word Chain Task

Create or update a file called `word_chain.txt` in the root directory.

## Rules

If the file doesn't exist, create it starting with the word "RUBY".

If it exists, add a new word that starts with the last letter of the previous word. Each word should be:
- A real English word
- Related to programming, technology, or Ruby when possible
- On a new line

## Example Chain
```
RUBY
YAML
LOGIC
CODE
EXECUTION
```

## Instructions

1. Read the last word in the file
2. Take the last letter of that word
3. Add a new word starting with that letter
4. Append it to the file on a new line

## Context

This creates an endless word chain where each run must find a new word starting with the last letter of the previous word. Since there are always more words available, this task will never run out of changes to make.