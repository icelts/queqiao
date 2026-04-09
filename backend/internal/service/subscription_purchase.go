package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	RechargeOrderSourceBalance              = "payment"
	RechargeOrderSourceSubscriptionPurchase = "subscription_purchase"
	subscriptionPurchaseCurrency            = "CNY"
)

var (
	ErrSubscriptionPurchaseUnavailable = infraerrors.BadRequest("SUBSCRIPTION_PURCHASE_UNAVAILABLE", "subscription is not available for purchase")
	ErrSubscriptionPurchaseInvalidMeta = infraerrors.BadRequest("SUBSCRIPTION_PURCHASE_INVALID_META", "invalid subscription purchase metadata")
)

type SubscriptionPurchaseOption struct {
	GroupID             int64             `json:"group_id"`
	GroupName           string            `json:"group_name"`
	Description         string            `json:"description"`
	Platform            string            `json:"platform"`
	ValidityDays        int               `json:"validity_days"`
	PurchaseEnabled     bool              `json:"purchase_enabled"`
	PurchasePrice       float64           `json:"purchase_price"`
	Currency            string            `json:"currency"`
	SortOrder           int               `json:"sort_order"`
	DailyLimitUSD       *float64          `json:"daily_limit_usd,omitempty"`
	WeeklyLimitUSD      *float64          `json:"weekly_limit_usd,omitempty"`
	MonthlyLimitUSD     *float64          `json:"monthly_limit_usd,omitempty"`
	CurrentSubscription *UserSubscription `json:"current_subscription,omitempty"`
	IsRenewal           bool              `json:"is_renewal"`
}

type SubscriptionPurchaseMetadata struct {
	GroupID       int64   `json:"group_id"`
	GroupName     string  `json:"group_name"`
	ValidityDays  int     `json:"validity_days"`
	PurchasePrice float64 `json:"purchase_price"`
	Currency      string  `json:"currency"`
}

func (s *SubscriptionService) ListPurchaseOptions(ctx context.Context, userID int64) ([]SubscriptionPurchaseOption, error) {
	groups, err := s.groupRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	subscriptions, err := s.userSubRepo.ListByUserID(ctx, userID)
	if err != nil && !errors.Is(err, ErrSubscriptionNotFound) {
		return nil, err
	}

	subscriptionByGroupID := make(map[int64]*UserSubscription, len(subscriptions))
	for i := range subscriptions {
		sub := subscriptions[i]
		if _, exists := subscriptionByGroupID[sub.GroupID]; exists {
			continue
		}
		copySub := sub
		subscriptionByGroupID[sub.GroupID] = &copySub
	}

	options := make([]SubscriptionPurchaseOption, 0)
	for i := range groups {
		group := groups[i]
		if !group.IsSubscriptionType() || !group.PurchaseEnabled || group.PurchasePrice == nil || *group.PurchasePrice <= 0 {
			continue
		}
		option := buildSubscriptionPurchaseOption(&group, subscriptionByGroupID[group.ID])
		options = append(options, option)
	}

	return options, nil
}

func (s *SubscriptionService) GetPurchaseOption(ctx context.Context, userID, groupID int64) (*SubscriptionPurchaseOption, error) {
	group, err := s.groupRepo.GetByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !group.IsActive() || !group.IsSubscriptionType() || !group.PurchaseEnabled || group.PurchasePrice == nil || *group.PurchasePrice <= 0 {
		return nil, ErrSubscriptionPurchaseUnavailable
	}

	var currentSubscription *UserSubscription
	if sub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, userID, groupID); err == nil {
		currentSubscription = sub
	} else if !errors.Is(err, ErrSubscriptionNotFound) {
		return nil, err
	}

	option := buildSubscriptionPurchaseOption(group, currentSubscription)
	return &option, nil
}

func (s *SubscriptionService) ReversePurchasedSubscription(ctx context.Context, userID, groupID int64, validityDays int, note string) error {
	if validityDays <= 0 {
		return nil
	}

	sub, err := s.userSubRepo.GetByUserIDAndGroupID(ctx, userID, groupID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return nil
		}
		return err
	}

	now := time.Now()
	targetExpiresAt := sub.ExpiresAt.AddDate(0, 0, -validityDays)
	if targetExpiresAt.After(now) {
		if err := s.userSubRepo.ExtendExpiry(ctx, sub.ID, targetExpiresAt); err != nil {
			return err
		}
		if err := s.userSubRepo.UpdateStatus(ctx, sub.ID, SubscriptionStatusActive); err != nil {
			return err
		}
	} else {
		if err := s.userSubRepo.ExtendExpiry(ctx, sub.ID, now); err != nil {
			return err
		}
		if err := s.userSubRepo.UpdateStatus(ctx, sub.ID, SubscriptionStatusExpired); err != nil {
			return err
		}
	}

	note = strings.TrimSpace(note)
	if note != "" {
		newNotes := strings.TrimSpace(sub.Notes)
		if newNotes != "" {
			newNotes += "\n"
		}
		newNotes += note
		if err := s.userSubRepo.UpdateNotes(ctx, sub.ID, newNotes); err != nil {
			return err
		}
	}

	s.InvalidateSubCache(userID, groupID)
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, userID, groupID)
	}

	return nil
}

func BuildSubscriptionPurchaseMetadata(option *SubscriptionPurchaseOption) (string, error) {
	if option == nil {
		return "", ErrSubscriptionPurchaseInvalidMeta
	}

	payload := SubscriptionPurchaseMetadata{
		GroupID:       option.GroupID,
		GroupName:     strings.TrimSpace(option.GroupName),
		ValidityDays:  normalizeGroupValidityDays(option.ValidityDays),
		PurchasePrice: roundMoney(option.PurchasePrice),
		Currency:      firstNonBlank(option.Currency, subscriptionPurchaseCurrency),
	}
	if payload.GroupID <= 0 || payload.PurchasePrice <= 0 {
		return "", ErrSubscriptionPurchaseInvalidMeta
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func ParseSubscriptionPurchaseMetadata(raw string) (*SubscriptionPurchaseMetadata, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ErrSubscriptionPurchaseInvalidMeta
	}

	var payload SubscriptionPurchaseMetadata
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, ErrSubscriptionPurchaseInvalidMeta
	}
	payload.GroupName = strings.TrimSpace(payload.GroupName)
	payload.Currency = firstNonBlank(strings.TrimSpace(payload.Currency), subscriptionPurchaseCurrency)
	payload.ValidityDays = normalizeGroupValidityDays(payload.ValidityDays)
	payload.PurchasePrice = roundMoney(payload.PurchasePrice)
	if payload.GroupID <= 0 || payload.PurchasePrice <= 0 {
		return nil, ErrSubscriptionPurchaseInvalidMeta
	}

	return &payload, nil
}

func buildSubscriptionPurchaseOption(group *Group, currentSubscription *UserSubscription) SubscriptionPurchaseOption {
	option := SubscriptionPurchaseOption{
		GroupID:         group.ID,
		GroupName:       strings.TrimSpace(group.Name),
		Description:     strings.TrimSpace(group.Description),
		Platform:        group.Platform,
		ValidityDays:    normalizeGroupValidityDays(group.DefaultValidityDays),
		PurchaseEnabled: group.PurchaseEnabled,
		Currency:        subscriptionPurchaseCurrency,
		SortOrder:       group.SortOrder,
		DailyLimitUSD:   group.DailyLimitUSD,
		WeeklyLimitUSD:  group.WeeklyLimitUSD,
		MonthlyLimitUSD: group.MonthlyLimitUSD,
	}
	if group.PurchasePrice != nil {
		option.PurchasePrice = roundMoney(*group.PurchasePrice)
	}
	if currentSubscription != nil {
		copySub := *currentSubscription
		option.CurrentSubscription = &copySub
		option.IsRenewal = true
	}
	return option
}
