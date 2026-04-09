-- 091: user-specific subscription purchase price overrides

CREATE TABLE IF NOT EXISTS user_subscription_purchase_prices (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    purchase_price DECIMAL(20,8) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_subscription_purchase_prices_user_group_unique UNIQUE (user_id, group_id),
    CONSTRAINT user_subscription_purchase_prices_purchase_price_positive CHECK (purchase_price > 0)
);

CREATE INDEX IF NOT EXISTS idx_user_subscription_purchase_prices_group_id
    ON user_subscription_purchase_prices(group_id);

CREATE INDEX IF NOT EXISTS idx_user_subscription_purchase_prices_user_id
    ON user_subscription_purchase_prices(user_id);

COMMENT ON TABLE user_subscription_purchase_prices IS '用户专属订阅购买价格配置';
COMMENT ON COLUMN user_subscription_purchase_prices.user_id IS '用户 ID';
COMMENT ON COLUMN user_subscription_purchase_prices.group_id IS '订阅分组 ID';
COMMENT ON COLUMN user_subscription_purchase_prices.purchase_price IS '用户专属订阅售价（CNY）';
