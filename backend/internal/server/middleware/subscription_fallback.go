package middleware

import (
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func isSubscriptionUsageLimitExceeded(err error) bool {
	return errors.Is(err, service.ErrDailyLimitExceeded) ||
		errors.Is(err, service.ErrWeeklyLimitExceeded) ||
		errors.Is(err, service.ErrMonthlyLimitExceeded)
}

func shouldFallbackToBalanceOnSubscriptionLimit(apiKey *service.APIKey, err error) bool {
	if !isSubscriptionUsageLimitExceeded(err) {
		return false
	}
	if apiKey == nil || apiKey.User == nil {
		return false
	}
	if !apiKey.User.SubscriptionLimitFallbackToBalance {
		return false
	}
	return apiKey.User.Balance > 0
}
