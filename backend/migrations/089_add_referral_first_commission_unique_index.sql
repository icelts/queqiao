CREATE UNIQUE INDEX IF NOT EXISTS idx_users_referral_code_active
	ON users(referral_code)
	WHERE deleted_at IS NULL AND referral_code <> '';

CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_commissions_first_once_per_referred
	ON referral_commissions(referred_user_id)
	WHERE commission_type = 'first';
