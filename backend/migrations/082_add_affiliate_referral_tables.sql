ALTER TABLE users
	ADD COLUMN IF NOT EXISTS inviter_id BIGINT NULL,
	ADD COLUMN IF NOT EXISTS referral_code VARCHAR(32) NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS custom_first_commission_rate DECIMAL(10,4) NULL,
	ADD COLUMN IF NOT EXISTS custom_recurring_commission_rate DECIMAL(10,4) NULL;

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conname = 'users_inviter_id_fkey'
	) THEN
		ALTER TABLE users
			ADD CONSTRAINT users_inviter_id_fkey
				FOREIGN KEY (inviter_id) REFERENCES users(id) ON DELETE SET NULL;
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_users_inviter_id ON users(inviter_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code_active
	ON users(referral_code)
	WHERE deleted_at IS NULL AND referral_code <> '';

CREATE TABLE IF NOT EXISTS recharge_orders (
	id BIGSERIAL PRIMARY KEY,
	user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	order_no VARCHAR(64) NOT NULL,
	external_order_id VARCHAR(128) NULL,
	channel VARCHAR(50) NOT NULL DEFAULT '',
	source VARCHAR(30) NOT NULL DEFAULT 'payment',
	currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
	amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	credited_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	status VARCHAR(20) NOT NULL DEFAULT 'pending',
	paid_at TIMESTAMPTZ NULL,
	refunded_at TIMESTAMPTZ NULL,
	callback_idempotency_key VARCHAR(128) NOT NULL DEFAULT '',
	callback_raw TEXT NULL,
	notes TEXT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT recharge_orders_order_no_key UNIQUE (order_no)
);

CREATE INDEX IF NOT EXISTS idx_recharge_orders_user_id ON recharge_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_status ON recharge_orders(status);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_paid_at ON recharge_orders(paid_at);
CREATE INDEX IF NOT EXISTS idx_recharge_orders_callback_idempotency_key ON recharge_orders(callback_idempotency_key);
CREATE UNIQUE INDEX IF NOT EXISTS idx_recharge_orders_channel_external_order_id
	ON recharge_orders(channel, external_order_id)
	WHERE external_order_id IS NOT NULL AND external_order_id <> '';

CREATE TABLE IF NOT EXISTS referral_commissions (
	id BIGSERIAL PRIMARY KEY,
	promoter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	referred_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	recharge_order_id BIGINT NOT NULL REFERENCES recharge_orders(id) ON DELETE CASCADE,
	commission_type VARCHAR(20) NOT NULL DEFAULT 'first',
	status VARCHAR(20) NOT NULL DEFAULT 'recorded',
	source_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	rate_snapshot DECIMAL(10,4) NOT NULL DEFAULT 0,
	commission_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	reversed_at TIMESTAMPTZ NULL,
	reversed_reason TEXT NULL,
	notes TEXT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_referral_commissions_promoter_user_id ON referral_commissions(promoter_user_id);
CREATE INDEX IF NOT EXISTS idx_referral_commissions_referred_user_id ON referral_commissions(referred_user_id);
CREATE INDEX IF NOT EXISTS idx_referral_commissions_recharge_order_id ON referral_commissions(recharge_order_id);
CREATE INDEX IF NOT EXISTS idx_referral_commissions_status ON referral_commissions(status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_commissions_order_type
	ON referral_commissions(recharge_order_id, commission_type);
