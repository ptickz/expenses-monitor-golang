-- name: IsConfigured :one
SELECT EXISTS (
    SELECT 1 FROM period_settings
    WHERE EXISTS (SELECT 1 FROM budget_settings)
);

-- name: CreatePeriodSettings :one
INSERT INTO period_settings (start_day, week_mode, time_zone, effective_from, created_by)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: AddPeriodSettingsUnion :exec
INSERT INTO period_settings_unions (period_settings_id, union_id)
VALUES ($1, $2);

-- name: CreateBudgetSettings :one
INSERT INTO budget_settings (weekly_amount, effective_from, created_by)
VALUES ($1, $2, $3)
RETURNING *;
