# Testing

## Backend unit tests

Run all tests that do not require PostgreSQL:

```powershell
go test ./...
```

These cover configuration validation, MAX init-data authentication, CORS, task utilities, WebSocket tickets, MAX API request contracts, and route/static-asset handling.

## Backend integration tests

Integration tests use a separate PostgreSQL database and never use production data.

1. Ensure the test database exists:

```powershell
docker exec kanban-postgres-1 psql -U kanban -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='kanban_test'"
```

2. Run API, webhook, and notification tests from a container attached to the Compose network:

```powershell
docker run --rm --network kanban_default -v "${PWD}:/app" -w /app -e TEST_DATABASE_URL=postgres://kanban:kanban@postgres:5432/kanban_test?sslmode=disable golang:1.26-alpine go test . -run "Test(APIEndToEnd|Webhook)"
docker run --rm --network kanban_default -v "${PWD}:/app" -w /app -e TEST_DATABASE_URL=postgres://kanban:kanban@postgres:5432/kanban_test?sslmode=disable golang:1.26-alpine go test ./services -run "TestNotifier|TestSend|TestAnswer|TestRegister|TestEnsure|TestLoadCustomCA"
```

Covered scenarios include:

- full organization, board, task, checklist, and comment lifecycle;
- custom statuses, reorder, archive, and restore;
- role promotion, last-owner protection, and cross-organization access denial;
- board-chat linking and authorization;
- task, status, assignment, and deadline notifications;
- MAX webhook idempotency and non-member rejection;
- official MAX message, callback, command, and subscription request shapes.

## Frontend tests

Run unit and component tests:

```powershell
npm test
```

Build production assets:

```powershell
npm run build
```

Covered scenarios include:

- organization settings and role visibility;
- board creation with custom statuses;
- board settings, status reorder/delete, and chat linking;
- task status changes and assignment;
- mobile tabs and swipe gestures;
- WebApp authentication behavior;
- task creation and duplicate-submission protection.

## Production smoke tests

After deployment, verify both direct backend access and the public reverse-proxy path:

```powershell
curl.exe -sS -D - --max-time 30 http://localhost:9300/max-kanban/api/health -o NUL
curl.exe -sS -D - --max-time 30 -o NUL https://kirillgvoz.ru/max-kanban/api/health
```

Also verify that the JavaScript bundle referenced by the served `index.html` returns JavaScript—not HTML—on both paths.
