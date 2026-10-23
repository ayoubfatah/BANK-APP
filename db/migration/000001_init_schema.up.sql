CREATE TABLE "users" (

  "id" bigserial PRIMARY KEY,

  "first_name" varchar NOT NULL,

  "last_name" varchar NOT NULL,

  "email" varchar UNIQUE NOT NULL,

  "role" varchar NOT NULL DEFAULT 'customer' CHECK (role IN ('customer', 'admin')),

  "img" varchar ,

  "phone" varchar UNIQUE,

  "password_hash" varchar NOT NULL,

  "status" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "updated_at" timestamptz NOT NULL DEFAULT (now())

);


CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash varchar  UNIQUE NOT NULL,
    is_blocked BOOLEAN NOT NULL DEFAULT FALSE,
    client_ip INET NOT NULL,
    user_agent TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE "kyc_verifications" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "document_type" varchar NOT NULL,

  "document_number" varchar,

  "status" varchar NOT NULL,

  "rejection_reason" text,

  "verified_at" timestamptz,

  "created_at" timestamptz NOT NULL DEFAULT (now())
  

);

CREATE TABLE "accounts" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "balance" bigint NOT NULL DEFAULT 0,

  "status" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "updated_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "account_details" (

  "id" bigserial PRIMARY KEY,

  "account_id" bigint NOT NULL,

  "type" varchar NOT NULL,

  "value" varchar NOT NULL,

  "country" varchar,

  "bank_name" varchar,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "entries" (

  "id" bigserial PRIMARY KEY,

  "account_id" bigint NOT NULL,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "type" varchar NOT NULL,

  "reference_id" bigint,

  "description" varchar,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "transfers" (

  "id" bigserial PRIMARY KEY,

  "from_account_id" bigint,

  "to_account_id" bigint,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "status" varchar NOT NULL,

  "reference" varchar,

  "idempotency_key" varchar UNIQUE,

  "failure_reason" text,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "completed_at" timestamptz

);

CREATE TABLE "fees" (

  "id" bigserial PRIMARY KEY,

  "transfer_id" bigint,

  "type" varchar NOT NULL,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "beneficiaries" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "name" varchar NOT NULL,

  "account_number" varchar,

  "iban" varchar,

  "bic" varchar,
  
  "routing_number" varchar,

  "bank_name" varchar,

  "country" varchar NOT NULL,

  "currency" varchar,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "updated_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "scheduled_transfers" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "from_account_id" bigint NOT NULL,

  "beneficiary_id" bigint NOT NULL,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "frequency" varchar NOT NULL,

  "next_run_at" timestamptz NOT NULL,

  "status" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "updated_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "currencies" (

  "code" varchar(3) PRIMARY KEY,

  "name" varchar NOT NULL,

  "symbol" varchar NOT NULL,

  "decimals" int NOT NULL DEFAULT 2,

  "is_active" boolean NOT NULL DEFAULT true,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "exchange_rates" (

  "id" bigserial PRIMARY KEY,

  "base_currency" varchar NOT NULL,

  "quote_currency" varchar NOT NULL,

  "rate" decimal(20,10) NOT NULL,

  "source" varchar
);

CREATE TABLE "conversions" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "from_account_id" bigint NOT NULL,

  "to_account_id" bigint NOT NULL,

  "from_amount" bigint NOT NULL,

  "to_amount" bigint NOT NULL,

  "from_currency" varchar NOT NULL,

  "to_currency" varchar NOT NULL,

  "exchange_rate" decimal(20,10) NOT NULL,

  "fee" bigint NOT NULL DEFAULT 0,

  "status" varchar NOT NULL,

  "idempotency_key" varchar UNIQUE,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "completed_at" timestamptz

);

CREATE TABLE "incoming_payments" (

  "id" bigserial PRIMARY KEY,

  "account_id" bigint NOT NULL,

  "sender_name" varchar,

  "sender_reference" varchar,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "status" varchar NOT NULL,

  "external_reference" varchar UNIQUE,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "completed_at" timestamptz

);


CREATE TABLE cards (
    id           bigserial PRIMARY KEY,
    user_id      bigint NOT NULL,
    account_id   bigint NOT NULL,

    card_number  varchar(19) NOT NULL
                 CHECK (card_number ~ '^[0-9]{13,19}$'),

    last_four    varchar(4) NOT NULL
                 CHECK (last_four ~ '^[0-9]{4}$'),

    cvv          varchar(4) NOT NULL
                 CHECK (cvv ~ '^[0-9]{3,4}$'),

    brand        varchar(20) NOT NULL
                 CHECK (brand IN (
                     'visa',
                     'mastercard',
                     'amex',
                     'discover'
                 )),

    card_type    varchar(10) NOT NULL DEFAULT 'physical'
                 CHECK (card_type IN (
                     'physical',
                     'virtual'
                 )),

    status       varchar(20) NOT NULL DEFAULT 'active'
                 CHECK (status IN (
                     'active',
                     'blocked',
                     'frozen',
                     'expired',
                     'cancelled'
                 )),

    cancelled_reason  varchar(20)
     CHECK (cancelled_reason IN ('lost', 'stolen', 'damaged', 'user_request', 'fraud')),

    expiry_date  date  NOT NULL,

    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    

    CHECK (right(card_number, 4) = last_four)
);




CREATE TABLE "card_transactions" (

  "id" bigserial PRIMARY KEY,

  "card_id" bigint NOT NULL,

  "account_id" bigint NOT NULL,

  "merchant_name" varchar NOT NULL,

  "merchant_category" varchar,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "status" varchar NOT NULL,

  "authorization_code" varchar,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "notifications" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint NOT NULL,

  "type" varchar NOT NULL,

  "title" varchar NOT NULL,

  "message" text NOT NULL,

  "read_at" timestamptz,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "audit_logs" (

  "id" bigserial PRIMARY KEY,

  "user_id" bigint,

  "action" varchar NOT NULL,

  "entity_type" varchar,

  "entity_id" bigint,

  "ip_address" varchar,

  "user_agent" text,

  "metadata" jsonb,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "businesses" (

  "id" bigserial PRIMARY KEY,

  "owner_id" bigint NOT NULL,

  "name" varchar NOT NULL,

  "registration_number" varchar,

  "country" varchar NOT NULL,

  "status" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now()),

  "updated_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "business_members" (

  "id" bigserial PRIMARY KEY,

  "business_id" bigint NOT NULL,

  "user_id" bigint NOT NULL,

  "role" varchar NOT NULL,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "invoices" (

  "id" bigserial PRIMARY KEY,

  "business_id" bigint NOT NULL,

  "customer_user_id" bigint,

  "customer_name" varchar NOT NULL,

  "customer_email" varchar,

  "amount" bigint NOT NULL,

  "currency" varchar NOT NULL,

  "status" varchar NOT NULL,

  "due_date" date,

  "paid_at" timestamptz,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);

CREATE TABLE "payment_links" (

  "id" bigserial PRIMARY KEY,

  "business_id" bigint NOT NULL,

  "token" varchar UNIQUE NOT NULL,

  "amount" bigint,

  "currency" varchar,

  "description" varchar,

  "status" varchar NOT NULL,

  "expires_at" timestamptz,

  "created_at" timestamptz NOT NULL DEFAULT (now())

);



-- ============================================================
-- INDEXES
-- ============================================================

CREATE INDEX ON "users" ("email");

CREATE INDEX ON "users" ("phone");

CREATE INDEX ON "users" ("role");


CREATE INDEX ON "users" ("status");

CREATE INDEX ON "refresh_tokens" ("user_id");

CREATE INDEX ON "refresh_tokens" ("token_hash");

CREATE INDEX ON "refresh_tokens" ("expires_at");

CREATE INDEX ON "kyc_verifications" ("user_id");

CREATE INDEX ON "kyc_verifications" ("status");

CREATE INDEX ON "accounts" ("user_id");

CREATE INDEX ON "accounts" ("currency");

CREATE INDEX ON "accounts" ("status");

CREATE UNIQUE INDEX ON "accounts" ("user_id", "currency");

CREATE INDEX ON "account_details" ("account_id");

CREATE UNIQUE INDEX ON "account_details" ("type", "value");

CREATE INDEX ON "entries" ("account_id");

CREATE INDEX ON "entries" ("type");

CREATE INDEX ON "entries" ("created_at");

CREATE INDEX ON "entries" ("account_id", "created_at");

CREATE INDEX ON "transfers" ("from_account_id");

CREATE INDEX ON "transfers" ("to_account_id");

CREATE INDEX ON "transfers" ("status");

CREATE INDEX ON "transfers" ("created_at");

CREATE INDEX ON "transfers" ("idempotency_key");

CREATE INDEX ON "fees" ("transfer_id");

CREATE INDEX ON "fees" ("type");

CREATE INDEX ON "beneficiaries" ("user_id");

CREATE INDEX ON "beneficiaries" ("iban");

CREATE INDEX ON "beneficiaries" ("account_number");

CREATE INDEX ON "scheduled_transfers" ("user_id");

CREATE INDEX ON "scheduled_transfers" ("from_account_id");

CREATE INDEX ON "scheduled_transfers" ("beneficiary_id");

CREATE INDEX ON "scheduled_transfers" ("next_run_at");

CREATE INDEX ON "scheduled_transfers" ("status");

CREATE INDEX ON "exchange_rates" ("base_currency", "quote_currency");



CREATE INDEX ON "conversions" ("user_id");


CREATE UNIQUE INDEX one_active_kyc_per_user
ON kyc_verifications (user_id)
WHERE status IN ('pending', 'verified');

CREATE INDEX ON "conversions" ("from_account_id");

CREATE INDEX ON "conversions" ("to_account_id");

CREATE INDEX ON "conversions" ("status");

CREATE INDEX ON "conversions" ("created_at");

CREATE INDEX ON "incoming_payments" ("account_id");

CREATE INDEX ON "incoming_payments" ("status");

CREATE INDEX ON "incoming_payments" ("created_at");

CREATE INDEX ON "cards" ("user_id");

CREATE INDEX ON "cards" ("account_id");

CREATE INDEX ON "cards" ("status");

CREATE INDEX ON "card_transactions" ("card_id");

CREATE INDEX ON "card_transactions" ("account_id");

CREATE INDEX ON "card_transactions" ("status");

CREATE INDEX ON "card_transactions" ("created_at");

CREATE INDEX ON "notifications" ("user_id");

CREATE INDEX ON "notifications" ("read_at");

CREATE INDEX ON "notifications" ("created_at");

CREATE INDEX ON "audit_logs" ("user_id");

CREATE INDEX ON "audit_logs" ("entity_type");

CREATE INDEX ON "audit_logs" ("entity_id");

CREATE INDEX ON "audit_logs" ("created_at");

CREATE INDEX ON "businesses" ("owner_id");

CREATE INDEX ON "businesses" ("registration_number");

CREATE UNIQUE INDEX ON "business_members" ("business_id", "user_id");

CREATE INDEX ON "business_members" ("business_id");

CREATE INDEX ON "business_members" ("user_id");

CREATE INDEX ON "invoices" ("business_id");

CREATE INDEX ON "invoices" ("customer_user_id");

CREATE INDEX ON "invoices" ("status");

CREATE INDEX ON "invoices" ("due_date");

CREATE INDEX ON "payment_links" ("business_id");

CREATE INDEX ON "payment_links" ("token");

CREATE INDEX ON "payment_links" ("status");


-- ============================================================
-- COMMENTS
-- ============================================================

COMMENT ON COLUMN "users"."status"
IS 'active, suspended, closed';

COMMENT ON COLUMN "kyc_verifications"."document_type"
IS 'passport, national_id, drivers_license';

COMMENT ON COLUMN "kyc_verifications"."status"
IS 'pending, verified, rejected';

COMMENT ON COLUMN "accounts"."status"
IS 'active, frozen, closed';

COMMENT ON COLUMN "account_details"."type"
IS 'iban, account_number, routing_number, sort_code';

COMMENT ON COLUMN "entries"."type"
IS 'transfer, conversion, deposit, withdrawal, card, fee, refund';

COMMENT ON COLUMN "transfers"."status"
IS 'pending, processing, completed, failed, cancelled';

COMMENT ON COLUMN "fees"."type"
IS 'transfer, conversion, card, withdrawal';

COMMENT ON COLUMN "scheduled_transfers"."frequency"
IS 'daily, weekly, monthly, yearly';

COMMENT ON COLUMN "scheduled_transfers"."status"
IS 'active, paused, cancelled';

COMMENT ON COLUMN "conversions"."status"
IS 'pending, completed, failed';

COMMENT ON COLUMN "incoming_payments"."status"
IS 'pending, completed, failed, reversed';

COMMENT ON COLUMN "cards"."card_type"
IS 'physical, virtual';

COMMENT ON COLUMN "cards"."status"
IS 'active, frozen, cancelled, expired';

COMMENT ON COLUMN "card_transactions"."status"
IS 'pending, completed, reversed, declined';

COMMENT ON COLUMN "businesses"."status"
IS 'active, suspended, closed';

COMMENT ON COLUMN "business_members"."role"
IS 'owner, admin, accountant, member';

COMMENT ON COLUMN "invoices"."status"
IS 'draft, sent, paid, overdue, cancelled';

COMMENT ON COLUMN "payment_links"."status"
IS 'active, paid, expired, cancelled';

COMMENT ON COLUMN "users"."role" IS 'customer, admin';

-- ============================================================
-- FOREIGN KEYS
-- ============================================================

-- USERS

ALTER TABLE "refresh_tokens"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "kyc_verifications"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "accounts"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "beneficiaries"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "scheduled_transfers"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "conversions"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "cards"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "notifications"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "audit_logs"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE SET NULL
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "businesses"
ADD FOREIGN KEY ("owner_id")
REFERENCES "users" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "business_members"
ADD FOREIGN KEY ("user_id")
REFERENCES "users" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;


-- ACCOUNTS

ALTER TABLE "account_details"
ADD FOREIGN KEY ("account_id")
REFERENCES "accounts" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "entries"
ADD FOREIGN KEY ("account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "transfers"
ADD FOREIGN KEY ("from_account_id")
REFERENCES "accounts" ("id")
ON DELETE SET NULL
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "transfers"
ADD FOREIGN KEY ("to_account_id")
REFERENCES "accounts" ("id")
ON DELETE SET NULL
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "scheduled_transfers"
ADD FOREIGN KEY ("from_account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "conversions"
ADD FOREIGN KEY ("from_account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "conversions"
ADD FOREIGN KEY ("to_account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "incoming_payments"
ADD FOREIGN KEY ("account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "cards"
ADD FOREIGN KEY ("account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "card_transactions"
ADD FOREIGN KEY ("account_id")
REFERENCES "accounts" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;


-- TRANSFERS

ALTER TABLE "fees"
ADD FOREIGN KEY ("transfer_id")
REFERENCES "transfers" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;


-- BENEFICIARIES

ALTER TABLE "scheduled_transfers"
ADD FOREIGN KEY ("beneficiary_id")
REFERENCES "beneficiaries" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;


-- CURRENCIES

ALTER TABLE "exchange_rates"
ADD FOREIGN KEY ("base_currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "exchange_rates"
ADD FOREIGN KEY ("quote_currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "conversions"
ADD FOREIGN KEY ("from_currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "conversions"
ADD FOREIGN KEY ("to_currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "invoices"
ADD FOREIGN KEY ("currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "payment_links"
ADD FOREIGN KEY ("currency")
REFERENCES "currencies" ("code")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;


-- CARDS

ALTER TABLE "card_transactions"
ADD FOREIGN KEY ("card_id")
REFERENCES "cards" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;


-- BUSINESSES

ALTER TABLE "business_members"
ADD FOREIGN KEY ("business_id")
REFERENCES "businesses" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "invoices"
ADD FOREIGN KEY ("business_id")
REFERENCES "businesses" ("id")
ON DELETE RESTRICT
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "invoices"
ADD FOREIGN KEY ("customer_user_id")
REFERENCES "users" ("id")
ON DELETE SET NULL
DEFERRABLE INITIALLY IMMEDIATE;

ALTER TABLE "payment_links"
ADD FOREIGN KEY ("business_id")
REFERENCES "businesses" ("id")
ON DELETE CASCADE
DEFERRABLE INITIALLY IMMEDIATE;
