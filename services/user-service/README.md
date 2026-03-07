# User Service

Handles user registration, authentication, and profile management.

## Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /signup | No | Register a new user |
| POST | /login | No | Login and get JWT token |
| GET | /health | No | Health check |
| GET | /users/me | Yes | Get current user profile |
| PUT | /users/me | Yes | Update profile (name, password) |

## Setup

1. Copy `.env.example` to `.env` and configure.
2. Run migrations: `psql $DATABASE_URL -f internal/database/migrations/001_init.sql -f internal/database/migrations/002_add_name.sql`
3. Run: `go run ./cmd`

## Docker

```bash
docker build -t user-service .
docker run -e DATABASE_URL=... -e JWT_SECRET=... -p 8080:8080 user-service
```
