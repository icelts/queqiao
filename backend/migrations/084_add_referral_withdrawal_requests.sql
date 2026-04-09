CREATE TABLE IF NOT EXISTS referral_withdrawal_requests (
	id BIGSERIAL PRIMARY KEY,
	promoter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	reviewer_user_id BIGINT NULL REFERENCES users(id) ON DELETE SET NULL,
	amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	currency VARCHAR(16) NOT NULL DEFAULT 'CNY',
	payment_method VARCHAR(32) NOT NULL DEFAULT '',
	account_name VARCHAR(100) NULL,
	account_identifier TEXT NULL,
	status VARCHAR(20) NOT NULL DEFAULT 'pending',
	reviewed_at TIMESTAMPTZ NULL,
	notes TEXT NULL,
	review_notes TEXT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_requests_promoter_user_id
	ON referral_withdrawal_requests(promoter_user_id);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_requests_reviewer_user_id
	ON referral_withdrawal_requests(reviewer_user_id);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_requests_status
	ON referral_withdrawal_requests(status);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_requests_created_at
	ON referral_withdrawal_requests(created_at);
