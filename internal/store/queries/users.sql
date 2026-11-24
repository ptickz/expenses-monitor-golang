-- name: UpsertUser :one
INSERT INTO users (source, external_id, name)
VALUES ($1, $2, $3)
ON CONFLICT (source, external_id) DO UPDATE SET name = EXCLUDED.name
RETURNING *;

-- name: GrantAdmin :exec
INSERT INTO admin_access (user_id)
VALUES ($1)
ON CONFLICT (user_id) DO NOTHING;

-- name: IsAdmin :one
SELECT EXISTS (SELECT 1 FROM admin_access WHERE user_id = $1);
