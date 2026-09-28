CREATE TABLE IF NOT EXISTS board_chats (
    id          BIGSERIAL PRIMARY KEY,
    board_id    BIGINT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    chat_id     BIGINT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    created_by  BIGINT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (board_id, chat_id)
);

CREATE TABLE IF NOT EXISTS webhook_events (
    id          BIGSERIAL PRIMARY KEY,
    event_key   TEXT UNIQUE NOT NULL,
    update_type TEXT NOT NULL,
    chat_id     BIGINT,
    user_id     BIGINT,
    callback_id TEXT,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    status      TEXT NOT NULL DEFAULT 'processed' CHECK (status IN ('processed','failed')),
    attempts    INTEGER NOT NULL DEFAULT 1 CHECK (attempts >= 0),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS notification_outbox (
    id              BIGSERIAL PRIMARY KEY,
    event_key       TEXT UNIQUE NOT NULL,
    chat_id         BIGINT,
    user_id         BIGINT,
    text            TEXT NOT NULL,
    payload         JSONB NOT NULL DEFAULT '{}'::jsonb,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
    attempts        INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_error      TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((chat_id IS NOT NULL AND user_id IS NULL) OR (chat_id IS NULL AND user_id IS NOT NULL))
);

CREATE INDEX IF NOT EXISTS idx_board_chats_board ON board_chats(board_id);
CREATE INDEX IF NOT EXISTS idx_board_chats_chat ON board_chats(chat_id);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_status ON notification_outbox(status, next_attempt_at, created_at);
CREATE INDEX IF NOT EXISTS idx_notification_outbox_chat ON notification_outbox(chat_id);
