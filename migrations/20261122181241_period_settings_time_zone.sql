-- +goose Up
ALTER TABLE "period_settings"
    ADD COLUMN "time_zone" TEXT NOT NULL DEFAULT 'Asia/Tbilisi',
    ADD CONSTRAINT "period_settings_time_zone_not_empty" CHECK ("time_zone" <> '');

ALTER TABLE "period_settings" ALTER COLUMN "time_zone" DROP DEFAULT;

-- +goose Down
ALTER TABLE "period_settings"
    DROP CONSTRAINT "period_settings_time_zone_not_empty",
    DROP COLUMN "time_zone";
