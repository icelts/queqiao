ALTER TABLE users
ADD COLUMN IF NOT EXISTS subscription_limit_fallback_to_balance BOOLEAN NOT NULL DEFAULT FALSE;
