ALTER TABLE referral_commissions
	ADD COLUMN IF NOT EXISTS currency VARCHAR(16) NOT NULL DEFAULT 'CNY';

UPDATE referral_commissions AS rc
SET currency = COALESCE(NULLIF(ro.currency, ''), 'CNY')
FROM recharge_orders AS ro
WHERE ro.id = rc.recharge_order_id
  AND rc.currency <> COALESCE(NULLIF(ro.currency, ''), 'CNY');
