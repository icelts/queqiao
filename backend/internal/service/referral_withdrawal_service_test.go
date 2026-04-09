//go:build unit

package service

import (
	"context"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalallocation"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalrequest"
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
	allocations, err := client.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.WithdrawalRequestIDEQ(item.ID)).
		All(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, allocations)
	allocationAmount := 0.0
	for _, entry := range allocations {
		allocationAmount = roundMoney(allocationAmount + entry.Amount)
	}
	require.InDelta(t, 5.0, allocationAmount, 1e-9)

	promoter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, promoter.Balance, 1e-9)

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
	require.ErrorIs(t, err, ErrReferralWithdrawalPendingExists)
}

func TestReferralService_CreateWithdrawalRequest_ConcurrentCallsDoNotOverAllocate(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "withdraw-concurrent-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("withdraw-concurrent-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-CONCURRENT-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	const workers = 2
	start := make(chan struct{})
	type result struct {
		item *ReferralWithdrawalRequest
		err  error
	}
	resultCh := make(chan result, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var item *ReferralWithdrawalRequest
			callErr := runWithSQLiteWriteRetry(func() error {
				var err error
				item, err = svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
					UserID:            promoterID,
					Amount:            4,
					PaymentMethod:     "alipay",
					AccountIdentifier: "alice@example",
				})
				return err
			})
			resultCh <- result{item: item, err: callErr}
		}()
	}

	close(start)
	wg.Wait()
	close(resultCh)

	successCount := 0
	for r := range resultCh {
		if r.err == nil {
			successCount++
			require.NotNil(t, r.item)
			require.Equal(t, ReferralWithdrawalStatusPending, r.item.Status)
			continue
		}
		require.ErrorIs(t, r.err, ErrReferralWithdrawalPendingExists)
	}
	require.Equal(t, 1, successCount)

	requests, err := client.ReferralWithdrawalRequest.Query().
		Where(
			referralwithdrawalrequest.PromoterUserIDEQ(promoterID),
			referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusPending),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.InDelta(t, 4.0, requests[0].Amount, 1e-9)

	allocations, err := client.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.PromoterUserIDEQ(promoterID)).
		All(ctx)
	require.NoError(t, err)
	allocationAmount := 0.0
	for _, entry := range allocations {
		allocationAmount = roundMoney(allocationAmount + entry.Amount)
	}
	require.InDelta(t, 4.0, allocationAmount, 1e-9)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 1.0, summary.AvailableCommission, 1e-9)
	require.InDelta(t, 4.0, summary.PendingWithdrawalAmount, 1e-9)
}

func TestReferralService_CreateWithdrawalRequest_RejectsWhenPendingRequestExists(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "withdraw-pending-limit-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("withdraw-pending-limit-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-PENDING-LIMIT-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	firstRequest, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            2,
		PaymentMethod:     "alipay",
		AccountIdentifier: "alice@example",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPending, firstRequest.Status)

	_, err = svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            1,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000009",
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalPendingExists)
}

func TestReferralService_ApproveWithdrawalRequest_IsIdempotentWithoutChangingBalance(t *testing.T) {
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
	require.InDelta(t, 0.0, promoter.Balance, 1e-9)
}

func TestReferralService_ApproveWithdrawalRequest_ConcurrentCallsApproveOnce(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "approve-concurrent-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("approve-concurrent-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-APPROVE-CONCURRENT-001",
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
		AccountIdentifier: "6222000000000001",
	})
	require.NoError(t, err)

	const workers = 8
	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			callErr := runWithSQLiteWriteRetry(func() error {
				_, err := svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
					RequestID:      item.ID,
					ReviewerUserID: promoterID,
					ReviewNotes:    "ok",
				})
				return err
			})
			errCh <- callErr
		}()
	}

	close(start)
	wg.Wait()
	close(errCh)
	for callErr := range errCh {
		require.NoError(t, callErr)
	}

	requestAfter, err := client.ReferralWithdrawalRequest.Get(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusApproved, requestAfter.Status)

	promoter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, promoter.Balance, 1e-9)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 2.0, summary.AvailableCommission, 1e-9)
	require.InDelta(t, 3.0, summary.ApprovedWithdrawalAmount, 1e-9)
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
	requestAfterRefund, err := client.ReferralWithdrawalRequest.Get(ctx, item.ID)
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusRejected, requestAfterRefund.Status)
	remainingAllocations, err := client.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.WithdrawalRequestIDEQ(item.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, remainingAllocations)

	_, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalStateInvalid)
}

func TestReferralService_MarkWithdrawalRequestPaid_SetsPaidState(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "mark-paid-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("mark-paid-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-MARK-PAID-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	item, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            4,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000002",
	})
	require.NoError(t, err)
	item, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
		ReviewNotes:    "approved",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusApproved, item.Status)

	item, err = svc.MarkWithdrawalRequestPaid(ctx, &MarkReferralWithdrawalPaidInput{
		RequestID:      item.ID,
		OperatorUserID: promoterID,
		PaymentNotes:   "paid",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPaid, item.Status)
	require.NotNil(t, item.PaidAt)

	item, err = svc.MarkWithdrawalRequestPaid(ctx, &MarkReferralWithdrawalPaidInput{
		RequestID:      item.ID,
		OperatorUserID: promoterID,
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPaid, item.Status)
}

func TestReferralService_PaidWithdrawalRefundCreatesDebtAndOffsetsFutureCommissions(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "true",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "50.0000",
		SettingKeyAffiliateWithdrawEnabled:       "true",
		SettingKeyAffiliateWithdrawMinAmount:     "1.00000000",
		SettingKeyAffiliateWithdrawMinInvitees:   "0",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "debt-promoter@test.com", 0)
	_, err := client.User.UpdateOneID(promoterID).
		SetRecurringCommissionEnabled(true).
		Save(ctx)
	require.NoError(t, err)
	referred, err := client.User.Create().
		SetEmail("debt-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-DEBT-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	withdrawal, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            5,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000003",
	})
	require.NoError(t, err)
	withdrawal, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      withdrawal.ID,
		ReviewerUserID: promoterID,
	})
	require.NoError(t, err)
	withdrawal, err = svc.MarkWithdrawalRequestPaid(ctx, &MarkReferralWithdrawalPaidInput{
		RequestID:      withdrawal.ID,
		OperatorUserID: promoterID,
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPaid, withdrawal.Status)

	_, _, err = svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
		OrderNo: "RC-WITHDRAW-DEBT-001",
		Channel: "manual",
	})
	require.NoError(t, err)

	promoterAfterRefund, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 5.0, promoterAfterRefund.ReferralWithdrawalDebt, 1e-9)
	_, err = svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            1,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000004",
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalDebtOutstanding)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-DEBT-002",
		Channel:        "manual",
		Amount:         6,
		CreditedAmount: 6,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	promoterAfterSecondRecharge, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 2.0, promoterAfterSecondRecharge.ReferralWithdrawalDebt, 1e-9)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-DEBT-003",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	promoterAfterThirdRecharge, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, promoterAfterThirdRecharge.ReferralWithdrawalDebt, 1e-9)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 3.0, summary.AvailableCommission, 1e-9)
	require.InDelta(t, 0.0, summary.WithdrawalDebt, 1e-9)
	require.True(t, summary.CanWithdraw)

	paidRequest, err := client.ReferralWithdrawalRequest.Get(ctx, withdrawal.ID)
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusPaid, paidRequest.Status)
	require.NotNil(t, paidRequest.PaidAt)
}

func TestReferralService_RejectWithdrawalRequest_AllowsApprovedAndReleasesAllocations(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "reject-approved-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("reject-approved-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-WITHDRAW-REJECT-APPROVED-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	ageAllReferralCommissions(t, ctx, client)

	item, err := svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            4,
		PaymentMethod:     "bank",
		AccountIdentifier: "6222000000000005",
	})
	require.NoError(t, err)
	item, err = svc.ApproveWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusApproved, item.Status)

	item, err = svc.RejectWithdrawalRequest(ctx, &ReviewReferralWithdrawalInput{
		RequestID:      item.ID,
		ReviewerUserID: promoterID,
		ReviewNotes:    "manual rollback",
	})
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusRejected, item.Status)
	allocations, err := client.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.WithdrawalRequestIDEQ(item.ID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, allocations)

	requestAfterReject, err := client.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.IDEQ(item.ID)).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, ReferralWithdrawalStatusRejected, requestAfterReject.Status)
}

func TestReferralService_CreateWithdrawalRequest_RejectsMixedCommissionCurrencies(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "mixed-currency-promoter@test.com", 0)
	referredCNY, err := client.User.Create().
		SetEmail("mixed-currency-cny@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)
	referredUSD, err := client.User.Create().
		SetEmail("mixed-currency-usd@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	orderCNY, err := client.RechargeOrder.Create().
		SetUserID(referredCNY.ID).
		SetOrderNo("RC-WITHDRAW-MIXED-CNY-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	orderUSD, err := client.RechargeOrder.Create().
		SetUserID(referredUSD.ID).
		SetOrderNo("RC-WITHDRAW-MIXED-USD-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("USD").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ReferralCommission.Create().
		SetPromoterUserID(promoterID).
		SetReferredUserID(referredCNY.ID).
		SetRechargeOrderID(orderCNY.ID).
		SetCommissionType(ReferralCommissionTypeFirst).
		SetStatus(ReferralCommissionStatusRecorded).
		SetSourceAmount(10).
		SetRateSnapshot(50).
		SetCommissionAmount(5).
		SetCurrency("CNY").
		Save(ctx)
	require.NoError(t, err)
	_, err = client.ReferralCommission.Create().
		SetPromoterUserID(promoterID).
		SetReferredUserID(referredUSD.ID).
		SetRechargeOrderID(orderUSD.ID).
		SetCommissionType(ReferralCommissionTypeRecurring).
		SetStatus(ReferralCommissionStatusRecorded).
		SetSourceAmount(10).
		SetRateSnapshot(50).
		SetCommissionAmount(5).
		SetCurrency("USD").
		Save(ctx)
	require.NoError(t, err)

	ageAllReferralCommissions(t, ctx, client)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.True(t, summary.HasMixedCommissionCurrencies)
	require.Empty(t, summary.CommissionCurrency)
	require.InDelta(t, 0.0, summary.AvailableCommission, 1e-9)
	require.False(t, summary.CanWithdraw)

	_, err = svc.CreateWithdrawalRequest(ctx, &CreateReferralWithdrawalInput{
		UserID:            promoterID,
		Amount:            1,
		PaymentMethod:     "alipay",
		AccountIdentifier: "mixed-currency-account",
	})
	require.ErrorIs(t, err, ErrReferralWithdrawalCurrencyConflict)
}

func TestReferralService_GetSummary_IncludesFrozenCommission(t *testing.T) {
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

	promoterID := mustCreateReferralTestUser(t, ctx, client, "summary-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("summary-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-SUMMARY-FROZEN-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)

	summary, err := svc.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, summary.AvailableCommission, 1e-9)
	require.InDelta(t, 5.0, summary.FrozenCommission, 1e-9)
	require.NotNil(t, summary.NextUnlockAt)
	require.True(t, summary.NextUnlockAt.After(time.Now()))
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
