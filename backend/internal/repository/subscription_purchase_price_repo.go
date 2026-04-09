package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type subscriptionPurchasePriceRepository struct {
	sql sqlExecutor
}

// NewSubscriptionPurchasePriceRepository creates a repository for
// user-specific subscription purchase price overrides.
func NewSubscriptionPurchasePriceRepository(sqlDB *sql.DB) service.SubscriptionPurchasePriceRepository {
	return &subscriptionPurchasePriceRepository{sql: sqlDB}
}

func (r *subscriptionPurchasePriceRepository) GetByUserID(ctx context.Context, userID int64) (map[int64]float64, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT group_id, purchase_price
		FROM user_subscription_purchase_prices
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	result := make(map[int64]float64)
	for rows.Next() {
		var groupID int64
		var purchasePrice float64
		if err := rows.Scan(&groupID, &purchasePrice); err != nil {
			return nil, err
		}
		result[groupID] = purchasePrice
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *subscriptionPurchasePriceRepository) GetByUserAndGroup(ctx context.Context, userID, groupID int64) (*float64, error) {
	var purchasePrice float64
	err := scanSingleRow(
		ctx,
		r.sql,
		`SELECT purchase_price FROM user_subscription_purchase_prices WHERE user_id = $1 AND group_id = $2`,
		[]any{userID, groupID},
		&purchasePrice,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &purchasePrice, nil
}

func (r *subscriptionPurchasePriceRepository) GetByGroupID(ctx context.Context, groupID int64) ([]service.UserSubscriptionPurchasePriceEntry, error) {
	rows, err := r.sql.QueryContext(ctx, `
		SELECT uspp.user_id, u.username, u.email, COALESCE(u.notes, ''), u.status, uspp.purchase_price
		FROM user_subscription_purchase_prices uspp
		JOIN users u ON u.id = uspp.user_id
		WHERE uspp.group_id = $1
		ORDER BY uspp.user_id
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var result []service.UserSubscriptionPurchasePriceEntry
	for rows.Next() {
		var entry service.UserSubscriptionPurchasePriceEntry
		if err := rows.Scan(
			&entry.UserID,
			&entry.UserName,
			&entry.UserEmail,
			&entry.UserNotes,
			&entry.UserStatus,
			&entry.PurchasePrice,
		); err != nil {
			return nil, err
		}
		result = append(result, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *subscriptionPurchasePriceRepository) SyncGroupPurchasePrices(ctx context.Context, groupID int64, entries []service.GroupSubscriptionPurchasePriceInput) error {
	if _, err := r.sql.ExecContext(ctx, `DELETE FROM user_subscription_purchase_prices WHERE group_id = $1`, groupID); err != nil {
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	userIDs := make([]int64, len(entries))
	purchasePrices := make([]float64, len(entries))
	for i, entry := range entries {
		userIDs[i] = entry.UserID
		purchasePrices[i] = entry.PurchasePrice
	}

	now := time.Now()
	_, err := r.sql.ExecContext(ctx, `
		INSERT INTO user_subscription_purchase_prices (user_id, group_id, purchase_price, created_at, updated_at)
		SELECT data.user_id, $1::bigint, data.purchase_price, $2::timestamptz, $2::timestamptz
		FROM unnest($3::bigint[], $4::double precision[]) AS data(user_id, purchase_price)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET
			purchase_price = EXCLUDED.purchase_price,
			updated_at = EXCLUDED.updated_at
	`, groupID, now, pq.Array(userIDs), pq.Array(purchasePrices))
	return err
}

func (r *subscriptionPurchasePriceRepository) DeleteByGroupID(ctx context.Context, groupID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM user_subscription_purchase_prices WHERE group_id = $1`, groupID)
	return err
}

func (r *subscriptionPurchasePriceRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM user_subscription_purchase_prices WHERE user_id = $1`, userID)
	return err
}
