# API reference

Both of these prefixes are supported:

- `/api/...`
- `/max-kanban/api/...`

Unless stated otherwise, API routes require a valid `X-Max-InitData` header.

## Authentication

### Validate MAX WebApp data

```text
POST /api/auth/validate
```

Request:

```json
{
  "initData": "query_id=...&user=...&hash=..."
}
```

The response contains the authenticated MAX `user_id` and display name.

### WebSocket ticket

```text
POST /api/ws-ticket
```

Request:

```json
{
  "org_id": 1
}
```

The caller must belong to the organization. The ticket is single-use.

## Organizations

| Method | Path | Access |
|---|---|---|
| `GET` | `/api/orgs` | Authenticated user; only organizations they belong to |
| `POST` | `/api/orgs` | Authenticated user; creator becomes owner |
| `GET` | `/api/orgs/:id` | Organization member |
| `PATCH` | `/api/orgs/:id` | Organization owner |
| `POST` | `/api/orgs/:id/members` | Organization owner |
| `DELETE` | `/api/orgs/:id/members/:uid` | Organization owner |

Member roles are `owner`, `admin`, and `member`. The final owner cannot be removed or demoted.

## Boards

| Method | Path | Access |
|---|---|---|
| `GET` | `/api/orgs/:id/boards` | Organization member |
| `GET` | `/api/orgs/:id/boards?include_archived=true` | Organization member |
| `POST` | `/api/orgs/:id/boards` | Organization admin or owner |
| `GET` | `/api/boards/:id` | Board organization member |
| `PATCH` | `/api/boards/:id` | Board organization admin or owner |
| `DELETE` | `/api/boards/:id` | Board organization admin or owner; archives the board |

Board creation accepts an optional custom `columns` array with names and `#RRGGBB` colors. A board may have 1–30 statuses.

## Statuses and columns

| Method | Path | Access |
|---|---|---|
| `POST` | `/api/boards/:id/columns` | Board organization admin or owner |
| `PATCH` | `/api/columns/:id` | Board organization admin or owner |
| `PATCH` | `/api/boards/:id/columns/reorder` | Board organization admin or owner |
| `DELETE` | `/api/columns/:id` | Board organization admin or owner |

Reorder requests must contain every column exactly once. Columns containing tasks cannot be deleted.

## Tasks

| Method | Path | Access |
|---|---|---|
| `GET` | `/api/boards/:id/tasks` | Board organization member |
| `POST` | `/api/boards/:id/tasks` | Board organization member |
| `GET` | `/api/tasks/:id` | Task organization member |
| `PATCH` | `/api/tasks/:id` | Task organization member |
| `PUT` | `/api/tasks/:id/move` | Task organization member |
| `DELETE` | `/api/tasks/:id` | Task organization member |
| `POST` | `/api/tasks/:id/assign` | Task organization member; assignee must belong to organization |
| `DELETE` | `/api/tasks/:id/assign/:uid` | Task organization member |

Task deadlines use `YYYY-MM-DD`. Sending explicit JSON `null` for `deadline` clears it.

## Checklists and comments

| Method | Path | Access |
|---|---|---|
| `POST` | `/api/tasks/:id/checklists` | Task organization member |
| `PATCH` | `/api/checklists/:id` | Task organization member |
| `DELETE` | `/api/checklists/:id` | Task organization member |
| `POST` | `/api/checklists/:id/items` | Task organization member |
| `PATCH` | `/api/checklist-items/:id` | Task organization member |
| `DELETE` | `/api/checklist-items/:id` | Task organization member |
| `GET` | `/api/tasks/:id/comments` | Task organization member |
| `POST` | `/api/tasks/:id/comments` | Task organization member |
| `DELETE` | `/api/comments/:id` | Comment author or organization admin/owner |

## Board chats

| Method | Path | Access |
|---|---|---|
| `GET` | `/api/boards/:id/chats` | Board organization admin or owner |
| `POST` | `/api/boards/:id/chats` | Board organization admin or owner |
| `DELETE` | `/api/board-chats/:id` | Board organization admin or owner |

## MAX webhook

```text
POST /webhook
POST /max-kanban/webhook
```

Webhook requests require `X-Max-Bot-Api-Secret`.

Supported MAX update types include:

- `message_created`
- `message_callback`
- `bot_started`
- `bot_added`

Supported working-chat commands include:

- `/link board_<id>`
- `/unlink board_<id>`
- `/new <task title>`
- `/tasks`
- `/start`

Callback payloads use:

- `take:<task_id>` to move a task to the next status
- `done:<task_id>` to move a task to the final status

Callback IDs are processed once. Duplicate deliveries receive an idempotent answer without repeating the task move.

## Health and metrics

```text
GET /api/health
GET /max-kanban/api/health
GET /api/metrics
GET /max-kanban/api/metrics
```

Health is unauthenticated. Metrics require authentication.

## Common errors

- `400`: invalid request data or business-rule violation.
- `401`: missing or invalid MAX authentication, invalid WebSocket ticket, or invalid webhook payload credentials.
- `403`: valid identity without sufficient role or membership.
- `404`: unknown resource, inaccessible organization, or unknown API route.
- `409`: unsafe or duplicate state, such as deleting a nonempty status or removing the final owner.
- `500`: operational failure without intentionally exposing database details.
- `503`: degraded health dependency.
