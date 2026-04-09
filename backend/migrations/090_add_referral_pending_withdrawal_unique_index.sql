CREATE UNIQUE INDEX IF NOT EXISTS idx_referral_withdrawal_requests_one_pending_per_promoter
	ON referral_withdrawal_requests(promoter_user_id)
	WHERE status = 'pending';
