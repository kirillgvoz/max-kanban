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
