package service

import "context"

// UserSubscriptionPurchasePriceEntry represents a user-specific subscription
// purchase price override inside a subscription group.
type UserSubscriptionPurchasePriceEntry struct {
	UserID        int64   `json:"user_id"`
	UserName      string  `json:"user_name"`
	UserEmail     string  `json:"user_email"`
	UserNotes     string  `json:"user_notes"`
	UserStatus    string  `json:"user_status"`
	PurchasePrice float64 `json:"purchase_price"`
}

// GroupSubscriptionPurchasePriceInput is used to batch sync user-specific
// purchase prices for a subscription group.
type GroupSubscriptionPurchasePriceInput struct {
	UserID        int64   `json:"user_id"`
	PurchasePrice float64 `json:"purchase_price"`
}

// SubscriptionPurchasePriceRepository stores user-specific purchase price
// overrides for subscription groups.
type SubscriptionPurchasePriceRepository interface {
	// GetByUserID returns all user-specific purchase price overrides keyed by group ID.
	GetByUserID(ctx context.Context, userID int64) (map[int64]float64, error)

	// GetByUserAndGroup returns the user-specific purchase price override for
	// the given user and group. Nil means no override exists.
	GetByUserAndGroup(ctx context.Context, userID, groupID int64) (*float64, error)

	// GetByGroupID lists all user-specific purchase price overrides configured
	// for the given subscription group.
	GetByGroupID(ctx context.Context, groupID int64) ([]UserSubscriptionPurchasePriceEntry, error)

	// SyncGroupPurchasePrices replaces the full set of purchase price overrides
	// for a subscription group.
	SyncGroupPurchasePrices(ctx context.Context, groupID int64, entries []GroupSubscriptionPurchasePriceInput) error

	// DeleteByGroupID removes all purchase price overrides for the given group.
	DeleteByGroupID(ctx context.Context, groupID int64) error

	// DeleteByUserID removes all purchase price overrides for the given user.
	DeleteByUserID(ctx context.Context, userID int64) error
}
