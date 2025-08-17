---
repository: ownerName/repoName
source: main
target: feature/timestamp-log
runType: run
title: Add timestamp log entry
---

# Timestamp Log Task

Create or update a file called `activity_log.txt` in the root directory.

## Instructions

Each run should add a new line to the file with the following format:
```
[YYYY-MM-DD HH:MM:SS] Run completed - Task ID: [random-uuid]
```

If the file doesn't exist, create it with a header:
```
=== RepoBird Activity Log ===
[first timestamp entry here]
```

Always append new entries to the end of the file. Generate a unique UUID for each task ID.

## Context

This task creates a continuously growing log file where each run adds a unique timestamp entry. Since each entry includes the current time and a unique ID, every run will always produce a unique change to the repository.