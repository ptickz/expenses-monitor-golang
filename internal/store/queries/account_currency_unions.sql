-- name: CreateUnion :one
INSERT INTO account_currency_unions (account_id, currency_id)
VALUES ($1, $2)
RETURNING *;

-- name: EnsureUnion :exec
INSERT INTO account_currency_unions (account_id, currency_id)
VALUES ($1, $2)
ON CONFLICT (account_id, currency_id) DO NOTHING;
