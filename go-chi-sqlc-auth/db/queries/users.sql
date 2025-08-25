-- name: CreateUser :one
INSERT INTO
    users (
        username,
        email,
        password_hash,
        first_name,
        last_name,
        phone_number,
        address,
        role
    )
VALUES (
        $1,
        $2,
        $3,
        $4,
        $5,
        $6,
        $7,
        $8
    )
RETURNING
    *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByUsername :one
SELECT * FROM users WHERE username = $1;

-- name: ListUsers :many
SELECT * FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET
    username = $2,
    email = $3,
    first_name = $4,
    last_name = $5,
    phone_number = $6,
    address = $7,
    role = $8,
    updated_at = now()
WHERE
    id = $1
RETURNING
    *;

-- name: UpdateUserPassword :one
UPDATE users
SET
    password_hash = $2,
    updated_at = now()
WHERE
    id = $1
RETURNING
    *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;