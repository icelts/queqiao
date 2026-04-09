//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReferralService_RecordPaidRecharge_RecurringCommissionRequiresPerUserEnablement(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "true",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "20.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "recurring-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("recurring-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, commissions, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-RECUR-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, commissions, 1)
	require.Equal(t, ReferralCommissionTypeFirst, commissions[0].CommissionType)

	_, commissions, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-RECUR-002",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, commissions, 0)

	_, err = client.User.UpdateOneID(promoterID).
		SetRecurringCommissionEnabled(true).
		Save(ctx)
	require.NoError(t, err)

	_, commissions, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-RECUR-003",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, commissions, 1)
	require.Equal(t, ReferralCommissionTypeRecurring, commissions[0].CommissionType)
	require.InDelta(t, 2.0, commissions[0].CommissionAmount, 1e-9)
}
