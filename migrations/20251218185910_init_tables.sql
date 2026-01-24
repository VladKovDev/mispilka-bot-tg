-- +goose Up
-- Таблица пользователей
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id BIGINT,
    username TEXT,
    first_name TEXT,
    last_name TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    blocked_at TIMESTAMP
);

-- Таблица телеграм-ботов
CREATE TABLE telegram_bots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bot_id BIGINT,
    username TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    "role" TEXT NOT NULL CHECK (role IN ('admin', 'general')),
    encrypted_token BYTEA NOT NULL,
    encryption_version INT NOT NULL,
    last_error TEXT,
    last_checked_at TIMESTAMP,
    disabled_at TIMESTAMP,
    revoked_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Связующая таблица пользователей и ботов (user - telegram_bot)
CREATE TABLE user_telegram_bots (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    telegram_bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'viewer')),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, telegram_bot_id)
);

-- Таблица скриптов
CREATE TABLE scripts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_bot_id UUID NOT NULL REFERENCES telegram_bots(id) ON DELETE CASCADE,
    "name" TEXT UNIQUE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    private_group_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Таблица сообщений
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    no_script BOOLEAN NOT NULL DEFAULT FALSE
);

-- Таблица шагов скриптов
CREATE TABLE script_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_id UUID NOT NULL REFERENCES scripts(id) ON DELETE CASCADE,
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE RESTRICT,
    "order" INT NOT NULL,
    channel TEXT NOT NULL,
    timing INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    skip_on_error BOOLEAN NOT NULL DEFAULT FALSE
);

-- Таблица прогресса пользователя по скриптам
CREATE TABLE script_progress (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE
    SET
        NULL,
        script_id UUID REFERENCES scripts(id) ON DELETE
    SET
        NULL,
        current_step_id UUID REFERENCES script_steps(id) ON DELETE
    SET
        NULL,
        "status" TEXT NOT NULL,
        step_started_at TIMESTAMP,
        started_at TIMESTAMP NOT NULL DEFAULT NOW(),
        finished_at TIMESTAMP
);

-- Таблица доставки сообщений шагов
CREATE TABLE script_progress_delivery (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID REFERENCES messages(id) ON DELETE
    SET
        NULL,
        step_id UUID REFERENCES script_steps(id) ON DELETE
    SET
        NULL,
        script_progress_id UUID REFERENCES script_progress(id) ON DELETE RESTRICT,
        sent_at TIMESTAMP,
        channel TEXT NOT NULL,
        "snapshot" JSONB NOT NULL,
        telegram_message_id TEXT NOT NULL
);

-- Таблица медиа файлов сообщений
CREATE TABLE message_media (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploaded_by UUID REFERENCES users(id) ON DELETE
    SET
        NULL,
        message_id UUID REFERENCES messages(id) ON DELETE
    SET
        NULL,
        storage_key TEXT NOT NULL,
        ext TEXT NOT NULL,
        "size" BIGINT NOT NULL,
        mime_type TEXT NOT NULL,
        created_at TIMESTAMP NOT NULL DEFAULT NOW(),
        deleted_at TIMESTAMP
);

-- Таблица кнопок сообщений
CREATE TABLE message_buttons (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    "text" TEXT NOT NULL,
    "url" TEXT,
    -- также сделать команду /command 
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

-- Таблица планирования шагов
CREATE TABLE scheduled_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    script_progress_id UUID NOT NULL REFERENCES script_progress(id) ON DELETE CASCADE,
    execute_at TIMESTAMP NOT NULL,
    "status" TEXT NOT NULL,
    send_at TIMESTAMP,
    sent_at TIMESTAMP,
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Таблица истории изменений
CREATE TABLE history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id UUID NOT NULL,
    entity_table TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    meta JSONB,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
-- Удаление таблиц
DROP TABLE IF EXISTS history;

DROP TABLE IF EXISTS scheduled_steps;

DROP TABLE IF EXISTS message_buttons;

DROP TABLE IF EXISTS message_media;

DROP TABLE IF EXISTS script_progress_delivery;

DROP TABLE IF EXISTS script_progress;

DROP TABLE IF EXISTS script_steps;

DROP TABLE IF EXISTS messages;

DROP TABLE IF EXISTS scripts;

DROP TABLE IF EXISTS user_telegram_bots;

DROP TABLE IF EXISTS telegram_bots;

DROP TABLE IF EXISTS users;