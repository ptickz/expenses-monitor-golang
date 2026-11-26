-- name: GetCurrencyByCode :one
SELECT * FROM currencies
WHERE code = $1;

-- name: ActivateCurrency :exec
UPDATE currencies
SET active = true
WHERE id = $1;
