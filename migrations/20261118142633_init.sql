-- +goose Up
CREATE TYPE "Direction" AS ENUM ('INCOME', 'OUTCOME');
CREATE TYPE "WeekMode" AS ENUM ('FROM_PERIOD_START', 'CALENDAR_WEEKS');
CREATE TYPE "ReasonBehavior" AS ENUM ('WEEKLY_EXPENSE', 'FREE_EXPENSE', 'GENERIC_EXPENSE', 'INCOME', 'SAVING', 'EXCHANGE');
CREATE TYPE "UserSource" AS ENUM ('TELEGRAM', 'HTTP');

-- Users and access

CREATE TABLE "users" (
    "id" SERIAL NOT NULL,
    "source" "UserSource" NOT NULL,
    "external_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "users_source_external_id_key" ON "users" ("source", "external_id");

CREATE TABLE "admin_access" (
    "user_id" INTEGER NOT NULL,
    "receives_alerts" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "admin_access_pkey" PRIMARY KEY ("user_id"),
    CONSTRAINT "admin_access_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Reference data

CREATE TABLE "payment_reasons" (
    "id" SERIAL NOT NULL,
    "title" TEXT NOT NULL,
    "behavior" "ReasonBehavior" NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,

    CONSTRAINT "payment_reasons_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "currencies" (
    "id" SERIAL NOT NULL,
    "code" TEXT NOT NULL,
    "active" BOOLEAN NOT NULL DEFAULT false,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "currencies_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "currencies_code_key" ON "currencies" ("code");

CREATE TABLE "countries" (
    "id" SERIAL NOT NULL,
    "title" TEXT NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,

    CONSTRAINT "countries_pkey" PRIMARY KEY ("id")
);

CREATE TABLE "accounts" (
    "id" SERIAL NOT NULL,
    "title" TEXT NOT NULL,
    "is_savings" BOOLEAN NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,

    CONSTRAINT "accounts_pkey" PRIMARY KEY ("id")
);
CREATE UNIQUE INDEX "accounts_single_savings_key" ON "accounts" ("is_savings") WHERE "is_savings";

CREATE TABLE "account_currency_unions" (
    "id" SERIAL NOT NULL,
    "account_id" INTEGER NOT NULL,
    "currency_id" INTEGER NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,

    CONSTRAINT "account_currency_unions_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "account_currency_unions_account_id_fkey" FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "account_currency_unions_currency_id_fkey" FOREIGN KEY ("currency_id") REFERENCES "currencies" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "account_currency_unions_account_id_currency_id_key" ON "account_currency_unions" ("account_id", "currency_id");

-- Operations

CREATE TABLE "operations" (
    "id" SERIAL NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "reason_id" INTEGER NOT NULL,
    "country_id" INTEGER,
    "comment" TEXT,
    "author_id" INTEGER NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,
    "deleted_by" INTEGER,

    CONSTRAINT "operations_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "operations_reason_id_fkey" FOREIGN KEY ("reason_id") REFERENCES "payment_reasons" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "operations_country_id_fkey" FOREIGN KEY ("country_id") REFERENCES "countries" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "operations_author_id_fkey" FOREIGN KEY ("author_id") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "operations_deleted_by_fkey" FOREIGN KEY ("deleted_by") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "operations_idempotency_key_key" ON "operations" ("idempotency_key");
CREATE INDEX "operations_created_at_idx" ON "operations" ("created_at");
CREATE INDEX "operations_author_id_idx" ON "operations" ("author_id");

CREATE TABLE "entries" (
    "id" SERIAL NOT NULL,
    "operation_id" INTEGER NOT NULL,
    "account_currency_union_id" INTEGER NOT NULL,
    "direction" "Direction" NOT NULL,
    "amount" BIGINT NOT NULL,

    CONSTRAINT "entries_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "entry_amount_positive" CHECK ("amount" > 0),
    CONSTRAINT "entries_operation_id_fkey" FOREIGN KEY ("operation_id") REFERENCES "operations" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "entries_account_currency_union_id_fkey" FOREIGN KEY ("account_currency_union_id") REFERENCES "account_currency_unions" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE INDEX "entries_operation_id_idx" ON "entries" ("operation_id");
CREATE INDEX "entries_account_currency_union_id_idx" ON "entries" ("account_currency_union_id");

-- Plans and borrowings

CREATE TABLE "plans" (
    "id" SERIAL NOT NULL,
    "account_currency_union_id" INTEGER NOT NULL,
    "reason_id" INTEGER NOT NULL,
    "amount" BIGINT NOT NULL,
    "comment" TEXT NOT NULL,
    "planned_on" DATE NOT NULL,
    "operation_id" INTEGER,
    "author_id" INTEGER NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,

    CONSTRAINT "plans_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "plan_amount_positive" CHECK ("amount" > 0),
    CONSTRAINT "plans_account_currency_union_id_fkey" FOREIGN KEY ("account_currency_union_id") REFERENCES "account_currency_unions" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "plans_reason_id_fkey" FOREIGN KEY ("reason_id") REFERENCES "payment_reasons" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "plans_operation_id_fkey" FOREIGN KEY ("operation_id") REFERENCES "operations" ("id") ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT "plans_author_id_fkey" FOREIGN KEY ("author_id") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);
CREATE UNIQUE INDEX "plans_operation_id_key" ON "plans" ("operation_id");

CREATE TABLE "week_borrowings" (
    "id" SERIAL NOT NULL,
    "from_week_start" DATE NOT NULL,
    "to_week_start" DATE NOT NULL,
    "amount" BIGINT NOT NULL,
    "author_id" INTEGER NOT NULL,
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "deleted_at" TIMESTAMPTZ,
    "deleted_by" INTEGER,

    CONSTRAINT "week_borrowings_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "week_borrowing_amount_positive" CHECK ("amount" > 0),
    CONSTRAINT "week_borrowings_author_id_fkey" FOREIGN KEY ("author_id") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "week_borrowings_deleted_by_fkey" FOREIGN KEY ("deleted_by") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- Settings

CREATE TABLE "budget_settings" (
    "id" SERIAL NOT NULL,
    "weekly_amount" BIGINT NOT NULL,
    "effective_from" TIMESTAMPTZ NOT NULL,
    "created_by" INTEGER NOT NULL,

    CONSTRAINT "budget_settings_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "budget_setting_weekly_amount_non_negative" CHECK ("weekly_amount" >= 0),
    CONSTRAINT "budget_settings_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE TABLE "period_settings" (
    "id" SERIAL NOT NULL,
    "start_day" INTEGER NOT NULL,
    "week_mode" "WeekMode" NOT NULL,
    "main_union_id" INTEGER NOT NULL,
    "effective_from" DATE NOT NULL,
    "created_by" INTEGER NOT NULL,

    CONSTRAINT "period_settings_pkey" PRIMARY KEY ("id"),
    CONSTRAINT "period_settings_start_day" CHECK ("start_day" BETWEEN 1 AND 31),
    CONSTRAINT "period_settings_main_union_id_fkey" FOREIGN KEY ("main_union_id") REFERENCES "account_currency_unions" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT "period_settings_created_by_fkey" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
);

-- +goose Down
DROP TABLE
    "period_settings",
    "budget_settings",
    "week_borrowings",
    "plans",
    "entries",
    "operations",
    "account_currency_unions",
    "accounts",
    "countries",
    "currencies",
    "payment_reasons",
    "admin_access",
    "users";

DROP TYPE "UserSource", "ReasonBehavior", "WeekMode", "Direction";
