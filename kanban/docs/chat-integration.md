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

## Mini-app attachment

Attach the mini-app URL in the partner panel
(business.max.ru → Чат-боты → ⋮ → Настройки → URL → Сохранить) so that
deep links (`https://max.ru/<bot>?startapp=…`) open the application.
The attached URL should match `FRONTEND_URL`.

Bot messages deliberately avoid `open_app` buttons: an unattached or
mismatched `web_app` URL makes MAX reject the whole message
(`404 Link not found`, `400 Field 'webApp' cannot be null`). Instead the bot
sends plain `link` buttons with board/task deep links — they need no
registered URL and always navigate somewhere. A one-shot `open_app` fallback
(retry without the button on link errors) is kept in the client as a safety
net.

Panel URL caveats: use `https`, at most 1024 characters; if the panel rejects
a URL with a path, serve the app from a subdomain root instead.

## Chat commands

Commands work in dialogs and group chats (the bot must be added to the
group). A trailing `@botname` mention, used in groups, is stripped, so
`/start@se14445725_bot` behaves like `/start`.

Group and channel `chat_id` values are negative (dialogs are positive);
the webhook accepts any non-zero chat id.

Non-text updates (stickers, media, channel posts) are acknowledged without a
reply; they never trigger webhook retries.

## Как устроен обмен с чатами (message journey)

### Подписка и вход

1. При старте backend выполняет `setupMaxIntegration` (`main.go`):
   `RegisterCommands` (start, tasks, new, link, unlink) и `EnsureSubscription`
   на URL из `MAX_WEBHOOK_URL` с типами из `MAX_WEBHOOK_UPDATE_TYPES`
   (по умолчанию `message_created,message_callback,bot_started,bot_added`).
2. MAX присылает события через `POST /webhook` (и дубль `/max-kanban/webhook`)
   с заголовком `X-Max-Bot-Api-Secret`.
3. `WebhookHandler.Handle` (`handlers/webhook.go`): сверка секрета, лимит тела
   1 МБ, разбор JSON (большие ID — через `json.Number`, потерь точности нет),
   маршрутизация по `update_type`.

### Входящее сообщение (`message_created` → `handleMessage`)

1. Извлекаются текст (`message.body.text`), автор (`sender.user_id`) и чат
   (`recipient.chat_id`).
2. Пустые/медийные апдейты без текста, автора или чата — постоянная ошибка:
   логируются и отвечают `200` без действий, чтобы MAX не слал бесполезные
   ретраи.
3. Событие записывается в `webhook_events` с уникальным `event_key`
   (повторные доставки MAX отбрасываются дедупликацией).
4. `parseChatCommand` делит текст на команду и аргументы, срезая хвостовое
   `@имябота` (нужно в группах): `/start@se14445725_bot` = `/start`,
   `/link@bot board_1` = `/link` + `board_1`.
5. Маршрут: `/link` → `linkBoard`, `/unlink` → `unlinkBoard`,
   `/new` → `createTaskFromChat`, `/start` и `/tasks` → `chatStatus`,
   остальное — подсказка по командам. Привязка/отвязка требуют роли
   `owner`/`admin` организации, создание задач — членства в организации.
6. Бизнес-логика выполняется в транзакции, затем commit, и только потом —
   ответ в чат. Дубль события (тот же `event_key`) делает commit и выходит
   без повторных эффектов.

### Ответы (`sendChatMessage` → `MaxBot.sendMessage`)

Каждый ответ идёт через `services/maxbot.go` и содержит кнопку `open_app`
с `web_app = FRONTEND_URL`:

1. Первая попытка — как есть (с кнопкой).
2. Если MAX отвечает `404 Link not found` или `400 Field 'webApp' cannot be
   null` (мини-приложение не привязано в панели или URL не совпал ровно),
   сообщение переотправляется **без** кнопки `open_app` — текст и остальные
   кнопки (`callback`, `link`) доходят всегда. В лог пишется
   `open_app button rejected ..., retrying without the mini-app button`.
3. Остальные ошибки (`chat.not.found`, `dialog.not.found`, `401`, `429`)
   возвращаются как есть — повтор бессмысленен или вреден.

Следствие: бот никогда не молчит из-за панели; кнопка «Открыть TaskFlow»
появляется сама, как только привязанный URL совпадёт с `FRONTEND_URL`.

### Кнопки (`message_callback` → `handleCallback`)

1. Payload `take:<task_id>` / `done:<task_id>` разбирается строго
   (`maxCallbackAction`); чужое — `200` без действий.
2. Дедуп по `callback_id`: повторный нажим отвечает «Действие уже
   обработано» и ничего не двигает.
3. Проверки: задача существует, нажавший — участник организации задачи.
   Промах — ответ в callback + сообщение в чат через `answerAndFail`.
4. `take` двигает задачу на следующий статус, `done` — на финальный
   (`moveTaskTx`), затем broadcast `task:moved` по WebSocket организации и
   постановка уведомления о статусе в outbox.

### Старт и добавление (`bot_started`, `bot_added` → `handleBotStarted`)

Запись события (дедуп) → `SendWelcome` с кнопкой `open_app` (с тем же
fallback). `bot_removed` и неизвестные типы — только лог и `200`.

### Уведомления (асинхронно, через outbox)

Создание задачи, смена статуса, назначение и дедлайны не зовут MAX API
напрямую, а кладут строки в `notification_outbox` с уникальным `event_key`
(`task-created:<task>:<chat>`, `task-status:...`, `task-assigned:<task>:<user>`,
`task-deadline:<task>:<user>:<day>`):

- воркер (`services.Notifier.Run`, тик 5 секунд) забирает по одной due-строке
  через `FOR UPDATE SKIP LOCKED` — безопасно при нескольких воркерах;
- отправляет через `MaxBot` (тот же fallback для кнопок, но уведомления —
  plain text и ему не подвержены);
- успех — `sent` + пауза 500 мс (соблюдение лимита MAX ~2 msg/s на диалог
  с запасом);
- ошибка — `pending` с backoff `attempts²` минут (потолок 1 час), после
  5 попыток — `failed` с текстом ошибки в `last_error`.

Дедлайны ставит планировщик раз в час (`runDeadlineScheduler` в `main.go`);
ключи содержат дату, поэтому повторы безопасны.

### Диалоги и группы

- `recipient.chat_type`: `dialog` (личка с ботом) или групповой чат/канал.
- Чтобы бот видел группу, его нужно добавить (прилетит `bot_added`).
  `chat_id` группы backend узнаёт только из входящих событий — списка чатов
  API MAX больше не отдаёт.
- В группах команды могут идти с `@упоминанием` — нормализация покрывает.
- Ответы уходят по `chat_id` события; прямые уведомления пользователю —
  по `user_id` (`SendDirectMessage`, требуется начатый диалог с ботом).

### Диплинки мини-приложения

Формат: `https://max.ru/<botName>?startapp=<payload>` (`BuildDeepLink`,
payload санитизируется под алфавит MAX, пустое имя бота даёт пустую строку).
Payload: только латиница, цифры, `_`, `-`, до 512 символов — более длинные
или с лишними символами MAX вычищает. Наши payload вида `board_<id>`,
`task_<id>` требованиям соответствуют.

Сообщения бота уже содержат диплинки: приветствие — ссылку на приложение,
`/link` — на доску (`board_<id>`), `/new` — на задачу (`task_<id>`).
Фронт читает payload при старте (`useStartParam`: сначала
`window.WebApp.initDataUnsafe.start_param`, запасной вариант — query-параметр
`?startapp=`) и один раз навигирует: `board_<id>` — сразу на доску,
`task_<id>` — через задачу на её доску (цепочка `tasks.get → boards.get →
orgs.get`). Неизвестный id или отсутствие доступа — молча остаёмся на
стартовом экране.

## Limitations

- Chat linking uses numeric MAX `chat_id` values; there is no native MAX chat picker yet.
- `/new` supports one linked board per chat.
- `/tasks` summarizes linked boards and upcoming deadlines rather than listing every assigned task.
- Chat notifications carry a compact two-line card (board, status, deadline, actor) with `take` and `done` buttons; direct user notifications stay text-only because dialog callbacks cannot resolve a chat.
- `/link` pins the board card message in the chat (bot must be admin); `/unlink` removes the pin only when it is still the bot’s own message. The pin id is stored in `board_chats.pinned_message_id`.
