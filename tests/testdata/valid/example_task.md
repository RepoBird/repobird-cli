---
prompt: Refactor the database connection module for better error handling
repository: backend/api-service
source: main
target: refactor/db-connection
runType: run
title: Database Connection Refactoring
context: The current database connection module lacks proper error handling and retry logic
files:
  - src/db/connection.go
  - src/db/pool.go
  - src/config/database.go
---

# Database Connection Refactoring Task

## Overview
This task involves refactoring our database connection module to improve error handling and add automatic retry logic with exponential backoff.

## Current Issues
1. No retry logic for transient failures
2. Poor error messages that don't help with debugging
3. Connection pool settings are hardcoded

## Requirements
- Implement exponential backoff for connection retries
- Add detailed error context using wrapped errors
- Make connection pool settings configurable
- Add comprehensive logging for debugging

## Success Criteria
- All database operations should retry transient failures automatically
- Error messages should include full context (operation, parameters, underlying error)
- Connection pool should be configurable via environment variables