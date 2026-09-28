# Architecture

## Runtime topology

```text
MAX messenger WebView
  React SPA under /max-kanban/
        |
        | HTTPS through Nginx
        v
Go/Gin backend on :9300
  REST API under /api and /max-kanban/api
  WebSocket under /ws and /max-kanban/ws
  MAX webhook under /webhook and /max-kanban/webhook
  Embedded production frontend under frontend/dist
        |
        v
PostgreSQL
```

Nginx may strip `/max-kanban` before proxying static requests. The backend therefore supports both prefixed and unprefixed frontend asset paths.

## Authentication and authorization

- MAX WebApp `initData` is validated with HMAC-SHA256.
- Validated identity is stored in the Gin request context as:
  - `user_id`
  - `username`
  - `display_name`
- Development bypass is allowed only when all three conditions are true:
  1. `DEV_MODE=1`;
  2. the request host is `localhost`, `127.0.0.1`, or `::1`;
  3. no usable production bypass is possible through the public Host header.
- Most object routes enforce organization membership.
- Board and column administration requires `admin` or `owner`.
- Organization settings and membership administration requires `owner`.
- Owners cannot normally be removed or demoted while they are the last remaining owner.
- Task assignees must already belong to the organization.

## Data model

High-level ownership is:

```text
organization
  org_member
  board
    column / status
    task
      checklist
      checklist item
      comment
      assignee
    board_chat
webhook_event
notification_outbox
schema_migration
```

Important integrity rules:

- Task columns must belong to the same board as the task.
- Positions are nonnegative integers.
- Column deletion is rejected while the column contains tasks.
- Board deletion archives the board rather than deleting its history.
- Notification event keys are unique to prevent duplicate notifications.

## Realtime updates

The backend exposes WebSocket broadcasts scoped by organization.

1. An authenticated client requests a short-lived ticket:

```text
POST /api/ws-ticket
{"org_id": 1}
```

2. The client opens:

```text
/ws?ticket=...
```

or

```text
/max-kanban/ws?ticket=...
```

3. Tickets are single-use and expire after 30 seconds.
4. Each room contains only clients for one organization.
5. Slow clients are disconnected rather than allowed to block broadcasting.

Representative event types:

- `board:created`, `board:updated`, `board:deleted`
- `column:created`, `column:updated`, `column:deleted`, `columns:reordered`
- `task:created`, `task:updated`, `task:moved`, `task:deleted`
- `task:assigned`, `task:unassigned`
- `checklist:created`, `checklist:updated`, `checklist:deleted`
- `checklist:item:created`, `checklist:item:updated`, `checklist:item:deleted`
- `comment:created`, `comment:deleted`

## Notifications

Task and status changes enqueue rows in `notification_outbox`, not direct MAX API calls.

A background worker:

- claims one due pending row at a time;
- sends it through the MAX Bot API;
- retries failed rows with backoff;
- marks rows `sent` or `failed`;
- paces sends to respect the documented MAX rate limit.

Deadline reminders are generated once per task, assignee, and calendar date.

## Database migrations

Migrations are embedded with `db/migrations/*.sql` and tracked in `schema_migrations`. Advisory locking prevents concurrent migration processes from applying the same migration twice.
