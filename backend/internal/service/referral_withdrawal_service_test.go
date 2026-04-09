//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/stretchr/testify/require"
)

func TestReferralService_CreateWithdrawalRequest_TracksPendingAmount(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
		SettingKeyAffiliateWithdrawEnabled:       "true",
		SettingKeyAffiliateWithdrawMinAmount:     "5.00000000",
		SettingKeyAffiliateWithdrawMinInvitees:   "1",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "withdraw-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("withdraw-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 5.0, summary.AvailableCommission, 1e-9)
	require.Equal(t, int64(1), summary.EffectiveInviteeCount)
	require.True(t, summary.CanWithdraw)

	item, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            5,
		PaymentMethod:     "alipay",
		AccountIdentifier: "alice@example",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPending, item.Status)

	promoter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 5.0, promoter.Balance, 1e-9)

	summary, err = svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, summary.AvailableCommission, 1e-9)
	require.InDelta(t, 5.0, summary.PendingWithdrawalAmount, 1e-9)
	require.False(t, summary.CanWithdraw)

	_, err = svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            1,
		PaymentMethod:     "alipay",
		AccountIdentifier: "alice@example",
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalThreshold)
}

func TestReferralService_ApproveWithdrawalRequest_DebitsBalanceOnce(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
		SettingKeyAffiliateWithdrawEnabled:       "true",
		SettingKeyAffiliateWithdrawMinAmount:     "1.00000000",
		SettingKeyAffiliateWithdrawMinInvitees:   "0",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "approve-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("approve-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-APPROVE-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	item, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            3,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000000",
	})
	require.NoError(t, err)

	item, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
		ReviewNotes:    "ok",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusApproved, item.Status)

	item, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusApproved, item.Status)

	promoter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 2.0, promoter.Balance, 1e-9)
}

func TestReferralService_ApproveWithdrawalRequest_BlockedAfterRefund(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
		SettingKeyAffiliateWithdrawEnabled:       "true",
		SettingKeyAffiliateWithdrawMinAmount:     "1.00000000",
		SettingKeyAffiliateWithdrawMinInvitees:   "0",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "refund-block-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("refund-block-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-REFUND-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	item, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            5,
		PaymentMethod:     "wechat",
		AccountIdentifier: "wxid_123",
	})
	require.NoError(t, err)

	_, _, err = svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
		OrderNo: "RC-WITHDRAW-REFUND-001",
		Channel: "manual",
	})
	require.NoError(t, err)

	_, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalInsufficient)

	item, err = svc.RejectWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
		ReviewNotes:    "refund happened",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusRejected, item.Status)
}

func ageAllReferralCommissions(t *testing.T, ctx context.Context, client *dbent.Client) {
	t.Helper()

	commissions, err := client.ReferralCommission.Query().All(ctx)
	require.NoError(t, err)

	maturedAt := time.Now().AddDate(0, -1, -1)
	for _, item := range commissions {
		_, err = client.ExecContext(
			ctx,
			"UPDATE referral_commissions SET created_at = ?, updated_at = ? WHERE id = ?",
			maturedAt,
			maturedAt,
			item.ID,
		)
		require.NoError(t, err)
	}
}
