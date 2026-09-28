# TaskFlow Kanban

## 1. Назначение решения

TaskFlow Kanban — чат-бот MAX с подключённым мини-приложением для управления задачами рабочей команды.

Решение предназначено для руководителя и участников малой проектной команды, которым нужно:

- вести задачи по статусам;
- назначать исполнителей;
- обсуждать работу в комментариях;
- связывать канбан-доску с рабочим чатом;
- получать уведомления MAX о новых задачах, смене статусов, назначениях и дедлайнах.

Каноническая реализация находится в каталоге `kanban/`.

Каталог `legacy/python-prototype/` содержит ранний прототип и не входит в поставку итогового решения.

## 2. Основной пользовательский сценарий

1. Пользователь открывает чат-бота TaskFlow в MAX.
2. Отправляет `/start`.
3. Открывает мини-приложение по кнопке «Открыть TaskFlow».
4. Создаёт организацию.
5. Создаёт доску со статусами.
6. Создаёт задачу.
7. Назначает исполнителя.
8. Перемещает задачу между статусами:
   - на десктопе — перетаскиванием карточки;
   - на мобильном устройстве — переключением вкладок и свайпом.
9. Привязывает доску к рабочему чату командой `/link board_<id>`.
10. Получает уведомления MAX при создании задачи, смене статуса, назначении и приближении дедлайна.

Создание задачи из рабочего чата также доступно командой:

```text
/new Подготовить отчёт
```

## 3. Состав и архитектура решения

```text
MAX WebView
  React + Vite + TypeScript
  MAX Bridge
        |
        | HTTPS через Nginx
        v
Go + Gin
  REST API: /api и /max-kanban/api
  MAX webhook: /webhook и /max-kanban/webhook
  WebSocket: /ws и /max-kanban/ws
  встроенный production-frontend
        |
        v
PostgreSQL
```

Ключевые backend-компоненты:

- `kanban/handlers/` — организации, доски, статусы, задачи, чеклисты, комментарии, связи с чатами, webhook;
- `kanban/middleware/` — проверка подписи MAX `initData` и RBAC;
- `kanban/services/` — клиент MAX Bot API и очередь уведомлений;
- `kanban/ws/` — realtime-обновления по организациям;
- `kanban/db/migrations/` — версионированная схема PostgreSQL;
- `kanban/frontend/` — мини-приложение.

Проверка подписи MAX выполняется по схеме HMAC-SHA256 `WebAppData`, без доверия данным клиента.

## 4. Одна команда для запуска всех локальных компонентов через Docker

Из корня репозитория:

```powershell
docker compose -f kanban/docker-compose.yml up --build -d
```

Команда запускает:

- PostgreSQL;
- backend TaskFlow;
- встроенный production-frontend backend.

Проверка после запуска:

```powershell
curl.exe http://localhost:9300/max-kanban/api/health
```

## 5. Необходимые параметры окружения

Перед запуском создать локальный файл окружения:

```powershell
Copy-Item kanban/.env.example kanban/.env
```

Обязательно задать собственные значения:

- `MAX_BOT_TOKEN`;
- `WEBHOOK_SECRET`;
- `DATABASE_URL` для production;
- `TEST_DATABASE_URL`, если запускаются интеграционные тесты;
- `MAX_WEBHOOK_URL`, если нужна автоматическая регистрация webhook;
- `MAX_CA_FILE`, если цепочка сертификатов требует дополнительного CA.

Пример секрета в репозитории использовать нельзя.

## 6. Переменные окружения

| Переменная | Назначение |
|---|---|
| `APP_ENV` | `development` или `production` |
| `APP_VERSION`, `GIT_SHA` | Версия и ревизия, отображаемые health-check |
| `PORT` | Порт backend, по умолчанию `9300` |
| `DATABASE_URL` | Строка подключения PostgreSQL |
| `MAX_BOT_TOKEN` | Токен MAX Bot API |
| `WEBHOOK_SECRET` | Секрет заголовка `X-Max-Bot-Api-Secret` |
| `FRONTEND_URL` | Публичный URL мини-приложения |
| `BOT_NAME` | Имя бота для диплинков |
| `MAX_WEBHOOK_URL` | Публичный HTTPS URL webhook |
| `MAX_WEBHOOK_UPDATE_TYPES` | События MAX для подписки |
| `MAX_CA_FILE` | Дополнительный CA для TLS MAX API |
| `ALLOWED_ORIGINS` | Разрешённые CORS-источники |
| `DEV_MODE` | Локальный обход auth, только localhost |
| `DEV_USER_ID` | Локальный тестовый пользователь |
| `TEST_DATABASE_URL` | Отдельная тестовая база PostgreSQL |

## 7. Используемые порты

| Порт | Назначение |
|---|---|
| `443` | Публичный HTTPS через Nginx |
| `9300` | Backend TaskFlow в Docker |
| `5432` | PostgreSQL в Docker |
| `3002` | Локальный Vite dev-server frontend |

## 8. Зависимости

Backend:

- `kanban/go.mod`
- `kanban/go.sum`
- Go `1.26.1`
- Gin `1.12.0`
- `gorilla/websocket 1.5.3`
- `pgx/v5 5.11.0`

Frontend:

- `kanban/frontend/package.json`
- `kanban/frontend/package-lock.json`
- React `19`
- Vite `6`
- TypeScript `5.7`
- Vitest `2.1`
- Testing Library и `jsdom`

Инфраструктура:

- `postgres:16-alpine`
- `node:20-alpine` на этапе сборки frontend
- `golang:1.26-alpine` на этапе сборки backend
- `alpine:3.19` в финальном образе

## 9. Внешние сервисы и интеграции

| Сервис | Назначение |
|---|---|
| MAX Bot API `https://platform-api2.max.ru` | Сообщения, кнопки, ответы на callback, команды, подписки webhook |
| Входящий MAX webhook | События `message_created`, `message_callback`, `bot_started`, `bot_added` |
| MAX Bridge `https://st.max.ru/js/max-web-app.js` | Данные запуска мини-приложения |
| Google Fonts | Шрифт интерфейса мини-приложения |
| PostgreSQL | Основное хранилище |
| Nginx | Публичный HTTPS reverse-proxy |

Интеграции с внешними бизнес-системами в MVP модельные и отсутствуют. Это явно зафиксировано как ограничение.

## 10. Описание работы с данными

PostgreSQL содержит:

- организации и роли;
- доски и статусы;
- задачи, назначения, чеклисты, комментарии;
- связи досок с рабочими чатами;
- идемпотентные события webhook;
- очередь уведомлений.

Миграции применяются автоматически при старте backend и защищены advisory lock.

Изменения задач рассылаются клиентам через WebSocket по организациям.

Уведомления сначала сохраняются в `notification_outbox`, затем отправляются фоновым воркером с повторными попытками.

Резервное копирование выполняется скриптами:

- `kanban/scripts/backup.sh`
- `kanban/scripts/restore.sh`

## 11. Порядок работы с тестовыми данными

Интеграционные backend-тесты используют только отдельную базу `kanban_test`.

Перед запуском убедиться, что база существует:

```powershell
docker exec kanban-postgres-1 psql -U kanban -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='kanban_test'"
```

Если базы нет, создать её:

```powershell
docker exec kanban-postgres-1 psql -U kanban -d postgres -c "CREATE DATABASE kanban_test"
```

Интеграционные тесты очищают тестовые таблицы и не используют production-данные.

Фикстуры API-проверок находятся в:

- `kanban/testdata/api-fixtures.json`

Синтетические тестовые `initData` для API-проверок генерируются локально:

```powershell
$env:MAX_BOT_TOKEN="<evaluation-bot-token>"
go run ./scripts/sign-test-initdata --user-id 9001 --username api-owner --display-name "API Owner"
```

Значение токена в репозиторий не записывается.

## 12. Пошаговый сценарий проверки

Backend unit-тесты:

```powershell
cd kanban
go test -count=1 ./...
```

Backend integration-тесты:

```powershell
docker compose up -d postgres
docker run --rm --network kanban_default -v "${PWD}:/app" -w /app -e TEST_DATABASE_URL=postgres://kanban:kanban@postgres:5432/kanban_test?sslmode=disable golang:1.26-alpine go test -count=1 . -run "Test(APIEndToEnd|Webhook)"
docker run --rm --network kanban_default -v "${PWD}:/app" -w /app -e TEST_DATABASE_URL=postgres://kanban:kanban@postgres:5432/kanban_test?sslmode=disable golang:1.26-alpine go test -count=1 ./services -run "TestNotifier|TestSend|TestAnswer|TestRegister|TestEnsure|TestLoadCustomCA"
```

Frontend unit-тесты:

```powershell
cd kanban/frontend
npm test
```

Production-сборка frontend:

```powershell
npm run build
```

Сборка Docker:

```powershell
cd kanban
docker compose build
```

Smoke-проверка локального backend:

```powershell
curl.exe http://localhost:9300/max-kanban/api/health
```

Smoke-проверка публичного контура:

```powershell
curl.exe https://kirillgvoz.ru/max-kanban/api/health
```

Сценарий MAX:

1. Открыть чат-бота.
2. Отправить `/start`.
3. Открыть мини-приложение.
4. Создать организацию и доску.
5. Создать задачу.
6. Переместить задачу и проверить уведомление в чате.
7. Нажать callback-кнопку и проверить смену статуса.

## 13. Примеры ожидаемого поведения системы

Успешное создание организации:

- HTTP `201`;
- ответ содержит `id`, `name`, `slug`;
- создатель получает роль `owner`.

Успешное создание задачи:

- HTTP `201`;
- ответ содержит `id`, `board_id`, `column_id`, `title`;
- уведомление ставится в очередь для связанных чатов.

Перемещение задачи:

- HTTP `200`;
- поле `column_id` соответствует новому статусу;
- подписчики организации получают WebSocket-событие `task:moved`;
- связанные чаты получают уведомление.

Повторный webhook callback:

- HTTP `200`;
- задача не перемещается повторно;
- пользователю возвращается ответ об уже обработанном действии.

Пользователь без членства в организации:

- HTTP `404` для объектов организации;
- HTTP `403` при недостатке роли.

Невалидный webhook-секрет:

- HTTP `403`;
- событие не обрабатывается.

## 14. Известные ограничения

- Ранний Python-прототип находится в `legacy/python-prototype/` и не поддерживается как продукт.
- Реальных интеграций с CRM, 1С и государственными системами нет.
- Webhook MAX требует публичного HTTPS на порту 443 и доверенного сертификата.
- Режим `DEV_MODE` предназначен только для localhost и запрещён в production.
- Автоматическая регистрация webhook выполняется только при корректных токене, URL, секрете и TLS.
- Презентационные рабочие секреты передаются отдельно и не хранятся в репозитории.
- Для Docker нужна запущенная Docker Desktop/Docker Engine.

## 15. Порядок остановки и повторного запуска решения

Остановить все контейнеры:

```powershell
docker compose -f kanban/docker-compose.yml down
```

Повторно запустить сохранённое состояние:

```powershell
docker compose -f kanban/docker-compose.yml up -d
```

Пересобрать backend после изменений:

```powershell
docker compose -f kanban/docker-compose.yml build backend
docker compose -f kanban/docker-compose.yml up -d backend
```

Проверить состояние:

```powershell
docker compose -f kanban/docker-compose.yml ps
docker compose -f kanban/docker-compose.yml logs backend --tail=100
```

Данные PostgreSQL сохраняются в Docker volume `pgdata`, пока volume явно не удалён.
