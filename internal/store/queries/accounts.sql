-- name: EnsureSavingsAccount :one
INSERT INTO accounts (title, is_savings)
VALUES ($1, true)
ON CONFLICT (is_savings) WHERE is_savings DO UPDATE SET title = accounts.title
RETURNING *;

-- name: CreateAccount :one
INSERT INTO accounts (title, is_savings)
VALUES ($1, false)
RETURNING *;
