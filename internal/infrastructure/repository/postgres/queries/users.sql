-- name: CreateUser :one
INSERT INTO
    users (
        telegram_id,
        username,
        first_name,
        last_name
    )
VALUES
    (
        @telegram_id,
        @username,
        @first_name,
        @last_name
    ) RETURNING id,
    telegram_id,
    username,
    first_name,
    last_name,
    created_at,
    is_active,
    blocked_at;

-- name: GetUserByID :one
SELECT
    id,
    telegram_id,
    username,
    first_name,
    last_name,
    created_at,
    is_active,
    blocked_at
FROM
    users
WHERE
    id = @id
    AND is_active = TRUE;

-- name: GetUserByTelegramID :one
SELECT
    id,
    telegram_id,
    username,
    first_name,
    last_name,
    created_at,
    is_active,
    blocked_at
FROM
    users
WHERE
    telegram_id = @telegram_id
    AND is_active = TRUE;

-- name: UpdateUser :one
UPDATE
    users
SET
    telegram_id = @telegram_id,
    username = @username,
    first_name = @first_name,
    last_name = @last_name
WHERE
    id = @id
    AND is_active = TRUE RETURNING id,
    telegram_id,
    username,
    first_name,
    last_name,
    created_at,
    is_active,
    blocked_at;

-- name: DeactivateUser :one
UPDATE
    users
SET
    is_active = FALSE,
    blocked_at = NOW()
WHERE
    id = @id
    AND is_active = TRUE RETURNING id;

-- name: DeleteUser :exec
DELETE FROM
    users
WHERE
    id = @id;

-- name: ListUsers :many
SELECT
    id,
    telegram_id,
    username,
    first_name,
    last_name,
    created_at,
    is_active,
    blocked_at
FROM
    users
WHERE
    is_active = TRUE
ORDER BY
    created_at DESC
LIMIT
    @limit_val OFFSET @offset_val;

-- name: CountUsers :one
SELECT
    COUNT(*)
FROM
    users
WHERE
    is_active = TRUE;

-- name: UserExistsByTelegramID :one
SELECT
    EXISTS(
        SELECT
            1
        FROM
            users
        WHERE
            telegram_id = @telegram_id
            AND is_active = TRUE
    );