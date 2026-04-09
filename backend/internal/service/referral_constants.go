package service

import "github.com/Wei-Shaw/sub2api/internal/domain"

const (
	RechargeOrderStatusPending  = domain.RechargeOrderStatusPending
	RechargeOrderStatusPaid     = domain.RechargeOrderStatusPaid
	RechargeOrderStatusFailed   = domain.RechargeOrderStatusFailed
	RechargeOrderStatusRefunded = domain.RechargeOrderStatusRefunded
)

const (
	ReferralCommissionTypeFirst     = domain.ReferralCommissionTypeFirst
	ReferralCommissionTypeRecurring = domain.ReferralCommissionTypeRecurring
)

const (
	ReferralCommissionStatusRecorded = domain.ReferralCommissionStatusRecorded
	ReferralCommissionStatusReversed = domain.ReferralCommissionStatusReversed
)

const (
	ReferralWithdrawalStatusPending  = domain.ReferralWithdrawalStatusPending
	ReferralWithdrawalStatusApproved = domain.ReferralWithdrawalStatusApproved
	ReferralWithdrawalStatusPaid     = domain.ReferralWithdrawalStatusPaid
	ReferralWithdrawalStatusRejected = domain.ReferralWithdrawalStatusRejected
	ReferralWithdrawalStatusCanceled = domain.ReferralWithdrawalStatusCanceled
)

const (
	SettingKeyAffiliateEnabled               = "affiliate_enabled"
	SettingKeyFirstCommissionEnabled         = "first_commission_enabled"
	SettingKeyRecurringCommissionEnabled     = "recurring_commission_enabled"
	SettingKeyDefaultFirstCommissionRate     = "default_first_commission_rate"
	SettingKeyDefaultRecurringCommissionRate = "default_recurring_commission_rate"
	SettingKeyAffiliateWithdrawEnabled       = "affiliate_withdraw_enabled"
	SettingKeyAffiliateWithdrawMinAmount     = "affiliate_withdraw_min_amount"
	SettingKeyAffiliateWithdrawMinInvitees   = "affiliate_withdraw_min_invitees"
)

const ReferralDefaultCurrency = "CNY"
