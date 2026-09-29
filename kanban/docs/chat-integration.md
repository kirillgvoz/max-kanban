# Chat integration

TaskFlow connects a MAX working chat to a Kanban board through the `board_chats`
relationship. The linked chat is used both as a command context and as a
notification destination.

## Integration flow

1. MAX sends bot events to:
   - `POST /webhook`
   - `POST /max-kanban/webhook`
2. Every webhook request must include `X-Max-Bot-Api-Secret`.
3. Supported updates are:
   - `message_created`
   - `message_callback`
   - `bot_started`
   - `bot_added`
4. On `bot_started` or `bot_added`, the bot sends a welcome message with a
   button that opens the TaskFlow mini-application.
5. An organization owner or admin links a board from the working chat.
   On mobile, the board ID is shown above the status tabs; the gear button
   opens **Board settings → Chats** with the ready-to-copy command:

   ```text
   /link board_123
   ```

6. The same role can unlink the board:

   ```text
   /unlink board_123
   ```

7. A board can also be linked manually from **Board settings → Chats** by
   entering the numeric MAX `chat_id`.
8. Only organization members can use linked-board commands. Linking and
   unlinking additionally require the `owner` or `admin` role.

## Working-chat commands

| Command | Behavior |
|---|---|
| `/start` | Sends welcome and board-linking instructions. |
| `/tasks` | Lists boards linked to the current chat and tasks with deadlines in the next seven days. |
| `/new <task title>` | Creates a task in the first status of the linked board. |
| `/link board_<id>` | Links the current chat to the specified active board. |
| `/unlink board_<id>` | Removes the link between the current chat and board. |

Any other message receives instructions for the supported commands.

`/new` has three important constraints:

- the chat must have at least one linked active board;
- the chat must have exactly one linked active board;
- the sender must belong to the board’s organization.

## Callback buttons

The webhook accepts callback payloads in these forms:

- `take:<task_id>`
- `done:<task_id>`

Behavior:

- `take` moves the task to the next configured status.
- `done` moves the task to the final configured status.
- the acting user must belong to the task’s organization;
- each callback ID is processed once;
- duplicate deliveries receive an idempotent answer without moving the task again.

## Notifications

Board-level events notify every chat linked to the affected board:

- task creation;
- task status changes.

User-level events notify the affected MAX user directly:

- task assignment;
- deadline on the current date.

Notifications use a persistent outbox. A background worker sends due
notifications, retries failures with backoff, respects the MAX rate limit,
and records pending, sent, and failed states.

Task changes are also broadcast to mini-application clients over an
organization-scoped WebSocket, so chat notifications and the board stay
consistent.

## Limitations

- Chat linking uses numeric MAX `chat_id` values; there is no native MAX chat picker yet.
- `/new` supports one linked board per chat.
- `/tasks` summarizes linked boards and upcoming deadlines rather than listing every assigned task.
- Production notifications currently emphasize text plus an “Open TaskFlow” button rather than attaching `take` and `done` buttons to every notification.
