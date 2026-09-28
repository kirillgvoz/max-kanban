# TaskFlow Kanban

TaskFlow is a MAX messenger mini-app for organizations, boards, customizable task statuses, tasks, checklists, comments, assignments, chat notifications, and deadline reminders.

## Repository layout

```text
kanban/
  main.go
  main_test.go
  main_webhook_test.go
  static_test.go
  config/
  db/migrations/
  handlers/
  middleware/
  models/
  services/
  ws/
  frontend/
  scripts/
  docs/
  Dockerfile
  docker-compose.yml
  .env.example
```

## Quick start

1. Copy the environment template:

```powershell
Copy-Item .env.example .env
```

2. Start PostgreSQL and the backend:

```powershell
docker compose up -d
```

3. Check health:

```powershell
curl.exe http://localhost:9300/max-kanban/api/health
```

The production frontend is embedded in the Go binary. Local frontend development uses Vite on port 3002.

## Test commands

Backend unit tests:

```powershell
go test ./...
```

Backend integration tests require a PostgreSQL test database:

```powershell
$env:TEST_DATABASE_URL = "postgres://kanban:kanban@postgres:5432/kanban_test?sslmode=disable"
docker run --rm --network kanban_default -v "${PWD}:/app" -w /app -e TEST_DATABASE_URL golang:1.26-alpine go test . -run "Test(APIEndToEnd|Webhook)"
```

Frontend tests and production build:

```powershell
npm test
npm run build
```

## Documentation

- `docs/architecture.md`
- `docs/api.md`
- `docs/operations.md`
- `docs/testing.md`
