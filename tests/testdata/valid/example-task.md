---
prompt: "Implement user authentication with JWT tokens and refresh token rotation"
repository: "acme/webapp"
source: "main"
target: "feature/jwt-auth"
runType: "run"
title: "Add JWT authentication system"
context: "The application needs secure authentication with JWT tokens including refresh token rotation for enhanced security"
files:
  - "src/auth/jwt.go"
  - "src/middleware/auth.go"
  - "src/models/user.go"
pullRequest:
  create: true
  draft: false
---

# JWT Authentication Implementation

## Overview
Implement a secure JWT-based authentication system with refresh token rotation to enhance application security.

## Requirements

### Authentication Flow
1. User login with credentials
2. Generate access token (15 min expiry) and refresh token (7 days expiry)
3. Store refresh token securely (hashed in database)
4. Implement token refresh endpoint
5. Automatic refresh token rotation on use

### Security Considerations
- Use RS256 algorithm for token signing
- Store private keys securely
- Implement rate limiting on auth endpoints
- Add token blacklisting for logout
- Log all authentication events

### API Endpoints
```
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
GET  /api/auth/verify
```

### Database Schema
```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES users(id)
);
```

## Implementation Notes
- Use existing user model structure
- Integrate with current middleware system
- Maintain backward compatibility with session-based auth during migration
- Add comprehensive test coverage for all auth scenarios

## Success Criteria
- [ ] All endpoints working correctly
- [ ] Token rotation functioning
- [ ] Security best practices implemented
- [ ] Tests passing with >80% coverage
- [ ] Documentation updated