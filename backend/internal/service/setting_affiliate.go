package service

import (
	"context"
	"strconv"
)

func (s *SettingService) IsAffiliateEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAffiliateEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsFirstCommissionEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyFirstCommissionEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsRecurringCommissionEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyRecurringCommissionEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) GetDefaultFirstCommissionRate(ctx context.Context) float64 {
	return s.getCommissionRate(ctx, SettingKeyDefaultFirstCommissionRate, 70)
}

func (s *SettingService) GetDefaultRecurringCommissionRate(ctx context.Context) float64 {
	return s.getCommissionRate(ctx, SettingKeyDefaultRecurringCommissionRate, 0)
}

func (s *SettingService) IsAffiliateWithdrawEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAffiliateWithdrawEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) GetAffiliateWithdrawMinAmount(ctx context.Context) float64 {
	return s.getCommissionRate(ctx, SettingKeyAffiliateWithdrawMinAmount, 100)
}

func (s *SettingService) GetAffiliateWithdrawMinInvitees(ctx context.Context) int64 {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAffiliateWithdrawMinInvitees)
	if err != nil {
		return 3
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed < 0 {
		return 3
	}
	return parsed
}

func (s *SettingService) getCommissionRate(ctx context.Context, key string, fallback float64) float64 {
	value, err := s.settingRepo.GetValue(ctx, key)
	if err != nil {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return normalizeCommissionRate(parsed)
}

func normalizeCommissionRate(rate float64) float64 {
	switch {
	case rate < 0:
		return 0
	case rate > 100:
		return 100
	default:
		return rate
	}
}
