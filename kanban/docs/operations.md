# Operations

## Environment variables

Copy `.env.example` to `.env` and provide deployment-specific values.

| Variable | Purpose |
|---|---|
| `APP_ENV` | `development` locally, `production` for deployment |
| `APP_VERSION`, `GIT_SHA` | Reported by `/api/health` |
| `PORT` | Backend listen port; default `9300` |
| `DATABASE_URL` | PostgreSQL connection string |
| `MAX_BOT_TOKEN` | Required MAX Bot API token |
| `WEBHOOK_SECRET` | Required `X-Max-Bot-Api-Secret`; must not use the example default in production |
| `FRONTEND_URL` | Public mini-app URL used by bot buttons |
| `BOT_NAME` | Bot name used for deep links |
| `MAX_WEBHOOK_URL` | Optional HTTPS webhook URL registered automatically at startup |
| `MAX_WEBHOOK_UPDATE_TYPES` | Optional webhook event list |
| `MAX_CA_FILE` | Optional extra trusted CA bundle for MAX API TLS |
| `ALLOWED_ORIGINS` | Exact browser origins permitted by CORS |
| `DEV_MODE` | Local development authentication bypass; never enable for public deployments |
| `DEV_USER_ID` | Optional local development user ID |
| `TEST_DATABASE_URL` | PostgreSQL URL used only by integration tests |

Production startup fails if `MAX_BOT_TOKEN` is missing, the webhook secret is missing or unchanged from the example, or the database URL still uses development credentials.

## Local deployment

```powershell
Copy-Item .env.example .env
docker compose up -d
curl.exe http://localhost:9300/max-kanban/api/health
```

To rebuild after code changes:

```powershell
docker compose build backend
docker compose up -d backend
```

## Nginx

The current production Nginx configuration proxies:

- `/max-kanban/api/` to backend `/api/`
- `/max-kanban/webhook` to backend `/webhook`
- `/max-kanban/ws` to backend `/ws` with WebSocket upgrade headers
- `/max-kanban/` to backend `/`, which serves the embedded SPA

Because the final mapping strips `/max-kanban`, backend static handling also supports unprefixed `/assets/...` paths.

After editing Nginx configuration, reload it from an administrator shell:

```powershell
cd C:\nginx
.\nginx.exe -s reload
```

## MAX webhook registration

The backend can register its webhook automatically when all of the following are configured:

- valid `MAX_BOT_TOKEN`;
- public HTTPS `MAX_WEBHOOK_URL`;
- approved `WEBHOOK_SECRET`;
- trusted TLS certificate chain.

MAX requires webhook delivery over HTTPS on port 443. If the platform cannot validate the certificate chain, add the required CA through `MAX_CA_FILE` and mount it into the backend container.

Manual registration example:

```powershell
curl.exe -X POST https://platform-api2.max.ru/subscriptions `
  -H "Authorization: <MAX_BOT_TOKEN>" `
  -H "Content-Type: application/json" `
  -d '{"url":"https://kirillgvoz.ru/max-kanban/webhook","update_types":["message_created","message_callback","bot_started","bot_added"],"secret":"<WEBHOOK_SECRET>"}'
```

## Database backups

Create a timestamped backup:

```sh
sh scripts/backup.sh
```

Restore requires explicit confirmation:

```sh
sh scripts/restore.sh backups/kanban-YYYYMMDDTHHMMSSZ.sql.gz
sh scripts/restore.sh --yes backups/kanban-YYYYMMDDTHHMMSSZ.sql.gz
```

The restore script overwrites the configured database. Verify the filename and environment before using `--yes`.

## Monitoring

Useful checks:

```powershell
curl.exe http://localhost:9300/max-kanban/api/health
curl.exe https://kirillgvoz.ru/max-kanban/api/health
docker compose ps
docker compose logs backend --tail=100
```

Authenticated operators can inspect:

```text
GET /api/metrics
```

Metrics include active WebSocket clients, pending/sent/failed notifications, and applied migration count.

## Troubleshooting

- `401` from the UI usually means MAX `initData` is missing or invalid. In production, do not work around this with `DEV_MODE`.
- Duplicate webhook deliveries are expected from MAX; callback processing and notification event keys are idempotent.
- If notifications remain `pending`, verify `MAX_BOT_TOKEN`, outbound HTTPS access, certificate trust, linked board chats, and MAX API rate limits.
- If a frontend asset returns HTML, verify the embedded production build and whether the reverse proxy stripped `/max-kanban`.
- If the bot stays silent in chats, check backend logs for `open_app button rejected`: the bot retried as plain text, so silence means a different error (`chat.not.found`, `401`, network). A `404 Link not found` alone is already handled by the fallback.
- `chat.not.found` / `dialog.not.found` on replies means the bot has no access to that chat: the user must start the bot (DM) or add it to the group first.

## Запуск с нуля (reproduce)

Полный путь от пустого сервера до говорящего бота. Шаги 1–2 — в кабинете
партнёра и на хосте, остальное — в репозитории.

### 1. Бот в кабинете партнёра

1. business.max.ru → профиль организации → раздел **Чат-боты** → создать бота,
   дождаться модерации.
2. В **⋮ → Настройки** бота скопировать токен → это `MAX_BOT_TOKEN`.
3. Там же позже (шаг 7) вставить URL мини-приложения в поле для ссылки и
   нажать **Сохранить**. URL должен **ровно** совпадать с `FRONTEND_URL`
   (включая завершающий слэш); панель принимает только `https`, до 1024
   символов, латиницу/цифры/точку/дефис. Если URL с путём отклоняется —
   вынеси приложение на корень поддомена.

### 2. Домен, TLS, Nginx

1. Домен с валидным HTTPS-сертификатом на 443 (MAX не ходит на другие порты
   и не принимает самоподписанные цепочки; промежуточные CA — через
   `MAX_CA_FILE`).
2. Проксировать на backend (`:9300`):
   - `/max-kanban/api/` → `/api/`;
   - `/max-kanban/webhook` → `/webhook`;
   - `/max-kanban/ws` → `/ws` с WebSocket-upgrade;
   - `/max-kanban/` → `/` (встроенный SPA).
3. `nginx -s reload`, проверить `https://<домен>/max-kanban/api/health`.

### 3. Конфигурация

```powershell
Copy-Item .env.example .env
```

Заполнить в `kanban/.env` (секреты не коммитить, `.env` в игноре):

- `APP_ENV=production` — включает строгую валидацию при старте;
- `MAX_BOT_TOKEN` — токен из шага 1;
- `WEBHOOK_SECRET` — длинная случайная строка (сверяется с заголовком
  `X-Max-Bot-Api-Secret`), не дефолт из примера;
- `DATABASE_URL` — боевые credentials, не `kanban:kanban`;
- `FRONTEND_URL` — ровно тот URL, что будет в панели (шаг 7);
- `BOT_NAME` — имя бота без `@`, для диплинков;
- `MAX_WEBHOOK_URL=https://<домен>/max-kanban/webhook` — иначе подписку
  придётся создавать вручную;
- `MAX_CA_FILE=./certs/russian-trusted-ca.pem` — если цепочке нужен
  российский CA (смонтирован в контейнер read-only);
- `ALLOWED_ORIGINS` — точные origin'ы фронта через запятую.

### 4. Запуск и сборка

```powershell
docker compose up -d
curl.exe http://localhost:9300/max-kanban/api/health
```

После любых правок Go или фронта:

```powershell
.\scripts\rebuild.ps1
```

Скрипт проверяет `.env`, маркеры свежего UI, пересобирает образ
(фронт вшит в бинарник через `go:embed`), ждёт `health ok + database ok`
и проверяет, что отдаётся свежий CSS-бандл. При битом docker-креденшеле
(`docker pull` падает с `error getting credentials`) — чинить
`~/.docker/config.json`, это вне репозитория.

### 5. Подписка вебхука

При старте backend регистрирует команды и подписку сам, если заданы
`MAX_BOT_TOKEN` + `MAX_WEBHOOK_URL`. Проверить:

```powershell
curl.exe https://platform-api2.max.ru/subscriptions -H "Authorization: <MAX_BOT_TOKEN>"
```

В ответе должны быть наш URL и типы
`message_created,message_callback,bot_started,bot_added`.
Ручная регистрация — командой из раздела «MAX webhook registration» выше.
Важно: `EnsureSubscription` не обновляет существующую подписку — при смене
URL или типов удали старую (`DELETE /subscriptions`) и дай backend
создать новую (или создай вручную).

### 6. Приёмка диалогов

1. Написать боту `/start` в личке → должен прийти текстовый ответ
   (с кнопкой — если шаг 7 выполнен, без — если нет, но текст всегда).
2. Повторный `/start`, `/tasks`, обычный текст, стикер: всё отвечает `200`
   на стороне вебхука, пользователь получает ответы на команды и подсказку
   на мусор.
3. В БД растут `webhook_events` (`SELECT count(*) FROM webhook_events`),
   в логах — `Server listening`, без `webhook ... error`.

Синтетическая проба без мессенджера (секрет из `.env`):

```powershell
curl.exe -X POST http://127.0.0.1:9300/webhook `
  -H "Content-Type: application/json" `
  -H "X-Max-Bot-Api-Secret: <WEBHOOK_SECRET>" `
  -d '{"update_type":"message_created","timestamp":1790000000000,"message":{"body":{"text":"/start"},"sender":{"user_id":1},"recipient":{"chat_id":99999,"chat_type":"dialog"}}}'
```

Ожидаемо `500` с `dialog.not.found` в логе (чат фейковый) — важно, что парсинг
и путь ответа отработали, а не `invalid message update`.

### 7. Привязка мини-приложения и групп

1. Панель → URL → Сохранить (шаг 1.3). Отправить `/start` ещё раз —
   теперь с кнопкой «Открыть TaskFlow».
2. Добавить бота в рабочую группу (прилетит `bot_added`), дать
   `/link board_<id>` от имени owner/admin → проверить `/tasks`, `/new Тест`,
   кнопки «Взять/Готово» на уведомлении о создании.
3. `GET /api/metrics` (авторизованно): `notification_outbox.pending`
   должен draining'иться в `sent`, `failed` — 0.

### 8. Бэкапы

```sh
sh scripts/backup.sh
sh scripts/restore.sh --yes backups/kanban-YYYYMMDDTHHMMSSZ.sql.gz
```

Restore перезаписывает БД — проверять имя файла и окружение дважды.
