-- +goose Up
CREATE UNIQUE INDEX "accounts_title_key" ON "accounts" (lower("title")) WHERE "deleted_at" IS NULL;

-- +goose Down
DROP INDEX "accounts_title_key";
