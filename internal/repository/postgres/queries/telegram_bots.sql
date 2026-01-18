-- name: CreateTelegramBot :one
INSERT INTO
    telegram_bots (
        owner_id,
        bot_id,
        username,
        first_name,
        encrypted_token,
        encryption_version,
        "status",
        last_error,
        last_checked_at,
        revoked_at
    )
VALUES
    (
        @owner_id,
        @bot_id,
        @username,
        @first_name,
        @encrypted_token,
        @encryption_version,
        @status,
        @last_error,
        @last_checked_at,
        @revoked_at
    ) RETURNING id,
    owner_id,
    bot_id,
    username,
    first_name,
    encrypted_token,
    encryption_version,
    "status",
    last_error,
    last_checked_at,
    revoked_at,
    created_at,
    updated_at;

-- name: GetTelegramBotByID :one
SELECT
    id,
    owner_id,
    bot_id,
    username,
    first_name,
    encrypted_token,
    encryption_version,
    "status",
    last_error,
    last_checked_at,
    revoked_at,
    created_at,
    updated_at
FROM
    telegram_bots
WHERE
    id = @id;

-- name: GetTelegramBotByBotID :one
SELECT
    id,
    owner_id,
    bot_id,
    username,
    first_name,
    encrypted_token,
    encryption_version,
    "status",
    last_error,
    last_checked_at,
    revoked_at,
    created_at,
    updated_at
FROM
    telegram_bots
WHERE
    bot_id = @bot_id;

-- name: UpdateTelegramBot :one
UPDATE
    telegram_bots
SET
    owner_id = @owner_id,
    bot_id = @bot_id,
    username = @username,
    first_name = @first_name,
    encrypted_token = @encrypted_token,
    encryption_version = @encryption_version,
    "status" = @status,
    last_error = @last_error,
    last_checked_at = @last_checked_at,
    revoked_at = @revoked_at,
    updated_at = NOW()
WHERE
    bot_id = @bot_id RETURNING id,
    owner_id,
    bot_id,
    username,
    first_name,
    encrypted_token,
    encryption_version,
    "status",
    last_error,
    last_checked_at,
    revoked_at,
    created_at,
    updated_at;

-- name: DeleteTelegramBot :exec
DELETE FROM
    telegram_bots
WHERE
    id = @id;

-- name: ListTelegramBots :many
SELECT
    id,
    owner_id,
    bot_id,
    username,
    first_name,
    encrypted_token,
    encryption_version,
    "status",
    last_error,
    last_checked_at,
    revoked_at,
    created_at,
    updated_at
FROM
    telegram_bots
ORDER BY
    created_at DESC;