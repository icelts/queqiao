package service

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID            int64
	Email         string
	Username      string
	Notes         string
	PasswordHash  string
	Role          string
	Balance       float64
	Concurrency   int
	Status        string
	AllowedGroups []int64
	TokenVersion  int64 // Incremented on password change to invalidate existing tokens
	CreatedAt     time.Time
	UpdatedAt     time.Time
	InviterID     *int64
	ReferralCode  string

	CustomFirstCommissionRate     *float64
	CustomRecurringCommissionRate *float64
	RecurringCommissionEnabled    bool

	// GroupRates maps group ID to user-specific rate multiplier overrides.
	GroupRates map[int64]float64

	// SubscriptionLimitFallbackToBalance controls whether requests can fall back to
	// wallet balance billing when subscription daily/weekly/monthly limits are exhausted.
	SubscriptionLimitFallbackToBalance bool

	SoraStorageQuotaBytes int64
	SoraStorageUsedBytes  int64
	TotpSecretEncrypted   *string
	TotpEnabled           bool
	TotpEnabledAt         *time.Time

	APIKeys       []APIKey
	Subscriptions []UserSubscription
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// CanBindGroup checks whether a user can bind to a given group.
// For standard groups:
// - Public groups (non-exclusive): all users can bind
// - Exclusive groups: only users with the group in AllowedGroups can bind
func (u *User) CanBindGroup(groupID int64, isExclusive bool) bool {
	if !isExclusive {
		return true
	}
	for _, id := range u.AllowedGroups {
		if id == groupID {
			return true
		}
	}
	return false
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}
