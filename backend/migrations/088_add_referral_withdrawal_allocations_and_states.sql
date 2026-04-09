ALTER TABLE referral_withdrawal_requests
	ADD COLUMN IF NOT EXISTS paid_at TIMESTAMPTZ NULL;

ALTER TABLE users
	ADD COLUMN IF NOT EXISTS has_successful_recharge BOOLEAN NOT NULL DEFAULT FALSE,
	ADD COLUMN IF NOT EXISTS referral_withdrawal_debt DECIMAL(20,8) NOT NULL DEFAULT 0;

UPDATE users AS u
SET has_successful_recharge = TRUE
WHERE EXISTS (
	SELECT 1
	FROM recharge_orders AS ro
	WHERE ro.user_id = u.id
	  AND ro.paid_at IS NOT NULL
);

CREATE TABLE IF NOT EXISTS referral_withdrawal_allocations (
	id BIGSERIAL PRIMARY KEY,
	promoter_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	withdrawal_request_id BIGINT NOT NULL REFERENCES referral_withdrawal_requests(id) ON DELETE CASCADE,
	commission_id BIGINT NOT NULL REFERENCES referral_commissions(id) ON DELETE CASCADE,
	amount DECIMAL(20,8) NOT NULL DEFAULT 0,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT referral_withdrawal_allocations_request_commission_key UNIQUE (withdrawal_request_id, commission_id)
);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_allocations_promoter_user_id
	ON referral_withdrawal_allocations(promoter_user_id);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_allocations_withdrawal_request_id
	ON referral_withdrawal_allocations(withdrawal_request_id);

CREATE INDEX IF NOT EXISTS idx_referral_withdrawal_allocations_commission_id
	ON referral_withdrawal_allocations(commission_id);
