//go:build unit

package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

func newReferralServiceSQLite(t *testing.T) (*ReferralService, *dbent.Client) {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	require.NoError(t, err)

	drv := entsql.OpenDB(dialect.SQLite, db)
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(drv)))
	t.Cleanup(func() { _ = client.Close() })

	return NewReferralService(client, nil, nil, nil, nil), client
}

func mustCreateReferralTestUser(t *testing.T, ctx context.Context, client *dbent.Client, email string, balance float64) int64 {
	t.Helper()
	u, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetBalance(balance).
		Save(ctx)
	require.NoError(t, err)
	return u.ID
}

func newReferralServiceSQLiteWithSettings(t *testing.T, settings map[string]string) (*ReferralService, *dbent.Client) {
	t.Helper()
	svc, client := newReferralServiceSQLite(t)
	if settings != nil {
		svc.settingService = NewSettingService(&settingRepoStub{values: settings}, &config.Config{})
	}
	return svc, client
}

func TestReferralService_RecordPaidRecharge_RefundedOrderRejected(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "record-refunded@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-REFUNDED-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusRefunded).
		SetRefundedAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         userID,
		OrderNo:        "RC-REFUNDED-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.ErrorIs(t, err, ErrRechargeOrderStateInvalid)

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(0), user.Balance)
}

func TestReferralService_CreateRechargeOrder_UsesFixedBalanceRechargeRatio(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyBalanceRechargeRatio: "100",
	})
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "fixed-ratio@test.com", 0)

	order, err := svc.CreateRechargeOrder(ctx, &CreateRechargeOrderInput{
		UserID:   userID,
		Amount:   10,
		Channel:  XunhuPayChannel,
		Currency: "CNY",
		Source:   RechargeOrderSourceBalance,
	})
	require.NoError(t, err)
	require.InDelta(t, 1000.0, order.CreditedAmount, 1e-9)
	require.InDelta(t, 10.0, order.Amount, 1e-9)
}

func TestReferralService_CreateRechargeOrder_NormalizesXunhuAmountPrecision(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyBalanceRechargeRatio: "100",
	})
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "xunhu-precision@test.com", 0)

	order, err := svc.CreateRechargeOrder(ctx, &CreateRechargeOrderInput{
		UserID:   userID,
		Amount:   10.004,
		Channel:  XunhuPayChannel,
		Currency: "CNY",
		Source:   RechargeOrderSourceBalance,
	})
	require.NoError(t, err)
	require.InDelta(t, 10.00, order.Amount, 1e-9)
	require.InDelta(t, 1000.0, order.CreditedAmount, 1e-9)
}

func TestReferralService_CreateRechargeOrder_RejectsNonCNYForXunhu(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyBalanceRechargeRatio: "100",
	})
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "xunhu-usd@test.com", 0)

	_, err := svc.CreateRechargeOrder(ctx, &CreateRechargeOrderInput{
		UserID:   userID,
		Amount:   10,
		Channel:  XunhuPayChannel,
		Currency: "USD",
		Source:   RechargeOrderSourceBalance,
	})
	require.ErrorIs(t, err, ErrRechargeOrderCurrencyUnsupported)
}

func TestReferralService_RecordPaidRecharge_PendingOrderCreditsOnce(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "record-pending@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-PENDING-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         userID,
		OrderNo:        "RC-PENDING-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         userID,
		OrderNo:        "RC-PENDING-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(10), user.Balance)

	orders, err := client.RechargeOrder.Query().Where(rechargeorder.OrderNoEQ("RC-PENDING-001")).All(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, RechargeOrderStatusPaid, orders[0].Status)
}

func TestReferralService_RecordPaidRecharge_RejectsCrossUserOrderRebind(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	ownerID := mustCreateReferralTestUser(t, ctx, client, "record-owner@test.com", 0)
	otherUserID := mustCreateReferralTestUser(t, ctx, client, "record-other@test.com", 0)
	order, err := client.RechargeOrder.Create().
		SetUserID(ownerID).
		SetOrderNo("RC-OWNER-ONLY-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         otherUserID,
		OrderNo:        order.OrderNo,
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.ErrorIs(t, err, ErrRechargeOrderUserMismatch)

	orderAfter, err := client.RechargeOrder.Get(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, ownerID, orderAfter.UserID)
	require.Equal(t, RechargeOrderStatusPending, orderAfter.Status)

	owner, err := client.User.Get(ctx, ownerID)
	require.NoError(t, err)
	require.Equal(t, float64(0), owner.Balance)

	other, err := client.User.Get(ctx, otherUserID)
	require.NoError(t, err)
	require.Equal(t, float64(0), other.Balance)
}

func TestReferralService_HandlePaymentWebhook_RejectsPaidAmountMismatch(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "webhook-mismatch-service@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-WEBHOOK-MISMATCH-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.HandlePaymentWebhook(ctx, &PaymentWebhookInput{
		OrderNo:  "RC-WEBHOOK-MISMATCH-001",
		Channel:  "manual",
		Status:   RechargeOrderStatusPaid,
		Amount:   8,
		Currency: "CNY",
	})
	require.ErrorIs(t, err, ErrRechargeOrderAmountMismatch)

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(0), user.Balance)

	order, err := client.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ("RC-WEBHOOK-MISMATCH-001")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusPending, order.Status)
}

func TestReferralService_HandlePaymentWebhook_PreservesOrderChannelAndCurrency(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "webhook-preserve-order@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-WEBHOOK-PRESERVE-001").
		SetChannel("custom").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	order, _, err := svc.HandlePaymentWebhook(ctx, &PaymentWebhookInput{
		OrderNo:        "RC-WEBHOOK-PRESERVE-001",
		Channel:        "malicious-channel",
		Status:         RechargeOrderStatusPaid,
		Amount:         10,
		Currency:       "USD",
		CreditedAmount: 10,
	})
	require.NoError(t, err)
	require.Equal(t, "custom", order.Channel)
	require.Equal(t, "CNY", order.Currency)

	entity, err := client.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ("RC-WEBHOOK-PRESERVE-001")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusPaid, entity.Status)
	require.Equal(t, "custom", entity.Channel)
	require.Equal(t, "CNY", entity.Currency)
}

func TestReferralService_RecordPaidRecharge_StoresCommissionCurrencyAsCNY(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "currency-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("currency-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, commissions, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-COMMISSION-CURRENCY-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, commissions, 1)
	require.Equal(t, "CNY", commissions[0].Currency)

	entity, err := client.ReferralCommission.Get(ctx, commissions[0].ID)
	require.NoError(t, err)
	require.Equal(t, "CNY", entity.Currency)
}

func TestReferralService_RecordPaidRecharge_SkipsCommissionForNonCNYOrder(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "noncny-promoter@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("noncny-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, commissions, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-COMMISSION-NONCNY-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "USD",
	})
	require.NoError(t, err)
	require.Len(t, commissions, 0)

	count, err := client.ReferralCommission.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}

func TestReferralService_RefundRechargeOrder_PaidOrderDebitsOnce(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "refund-once@test.com", 20)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-PAID-REFUND-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(5).
		SetCreditedAmount(5).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
		OrderNo: "RC-PAID-REFUND-001",
		Channel: "manual",
	})
	require.NoError(t, err)

	_, _, err = svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
		OrderNo: "RC-PAID-REFUND-001",
		Channel: "manual",
	})
	require.NoError(t, err)

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(15), user.Balance)
}

func TestReferralService_MarkRechargeOrderFailed_PendingOrderIdempotent(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "mark-failed-idempotent@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-FAILED-IDEMPOTENT-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	order, err := svc.MarkRechargeOrderFailed(ctx, "RC-FAILED-IDEMPOTENT-001", "manual", "", "cbk-1", `{"status":"failed"}`, "failed by gateway")
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusFailed, order.Status)

	order, err = svc.MarkRechargeOrderFailed(ctx, "RC-FAILED-IDEMPOTENT-001", "manual", "", "cbk-1", `{"status":"failed"}`, "duplicate failed callback")
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusFailed, order.Status)

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(0), user.Balance)
}

func TestReferralService_markRechargeOrderFailedOnPending_DoesNotOverridePaid(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "mark-failed-stale@test.com", 0)
	staleOrder, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-FAILED-STALE-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.RechargeOrder.UpdateOneID(staleOrder.ID).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.markRechargeOrderFailedOnPending(ctx, staleOrder, "cbk-race", `{"status":"failed"}`, "stale failed callback")
	require.ErrorIs(t, err, ErrRechargeOrderStateInvalid)

	orderAfter, err := client.RechargeOrder.Get(ctx, staleOrder.ID)
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusPaid, orderAfter.Status)
	require.Empty(t, orderAfter.CallbackIdempotencyKey)
}

func TestReferralService_RecordPaidRecharge_ConcurrentCallsCreditOnce(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "record-concurrent@test.com", 0)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-CONCURRENT-PAID-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
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
				_, _, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
					UserID:         userID,
					OrderNo:        "RC-CONCURRENT-PAID-001",
					Channel:        "manual",
					Amount:         10,
					CreditedAmount: 10,
					Currency:       "CNY",
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

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(10), user.Balance)

	order, err := client.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ("RC-CONCURRENT-PAID-001")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusPaid, order.Status)
}

func TestReferralService_RefundRechargeOrder_ConcurrentCallsDebitOnce(t *testing.T) {
	svc, client := newReferralServiceSQLite(t)
	ctx := context.Background()

	userID := mustCreateReferralTestUser(t, ctx, client, "refund-concurrent@test.com", 20)
	_, err := client.RechargeOrder.Create().
		SetUserID(userID).
		SetOrderNo("RC-CONCURRENT-REFUND-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(5).
		SetCreditedAmount(5).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(time.Now()).
		Save(ctx)
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
				_, _, err := svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
					OrderNo: "RC-CONCURRENT-REFUND-001",
					Channel: "manual",
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

	user, err := client.User.Get(ctx, userID)
	require.NoError(t, err)
	require.Equal(t, float64(15), user.Balance)

	order, err := client.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ("RC-CONCURRENT-REFUND-001")).
		Only(ctx)
	require.NoError(t, err)
	require.Equal(t, RechargeOrderStatusRefunded, order.Status)
}

func TestReferralService_RecordPaidRecharge_ConcurrentCallsCreateOneCommission(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "promoter-concurrent@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("referred-concurrent@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	order, err := client.RechargeOrder.Create().
		SetUserID(referred.ID).
		SetOrderNo("RC-CONCURRENT-COMMISSION-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
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
				_, _, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
					UserID:         referred.ID,
					OrderNo:        order.OrderNo,
					Channel:        "manual",
					Amount:         10,
					CreditedAmount: 10,
					Currency:       "CNY",
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

	referredAfter, err := client.User.Get(ctx, referred.ID)
	require.NoError(t, err)
	require.InDelta(t, 10.0, referredAfter.Balance, 1e-9)

	promoterAfter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, promoterAfter.Balance, 1e-9)

	commissions, err := client.ReferralCommission.Query().
		Where(referralcommission.RechargeOrderIDEQ(order.ID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commissions, 1)
	require.Equal(t, ReferralCommissionStatusRecorded, commissions[0].Status)
	require.Equal(t, ReferralCommissionTypeFirst, commissions[0].CommissionType)
	require.InDelta(t, 5.0, commissions[0].CommissionAmount, 1e-9)
}

func TestReferralService_RecordPaidRecharge_ConcurrentDifferentOrdersOnlyOneFirstCommission(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "true",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "20.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "promoter-concurrent-orders@test.com", 0)
	_, err := client.User.UpdateOneID(promoterID).SetRecurringCommissionEnabled(true).Save(ctx)
	require.NoError(t, err)

	referred, err := client.User.Create().
		SetEmail("referred-concurrent-orders@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.RechargeOrder.Create().
		SetUserID(referred.ID).
		SetOrderNo("RC-CONCURRENT-ORDER-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.RechargeOrder.Create().
		SetUserID(referred.ID).
		SetOrderNo("RC-CONCURRENT-ORDER-002").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	orderNos := []string{"RC-CONCURRENT-ORDER-001", "RC-CONCURRENT-ORDER-002"}
	const workers = 2
	start := make(chan struct{})
	errCh := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		orderNo := orderNos[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			callErr := runWithSQLiteWriteRetry(func() error {
				_, _, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
					UserID:         referred.ID,
					OrderNo:        orderNo,
					Channel:        "manual",
					Amount:         10,
					CreditedAmount: 10,
					Currency:       "CNY",
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

	referredAfter, err := client.User.Get(ctx, referred.ID)
	require.NoError(t, err)
	require.InDelta(t, 20.0, referredAfter.Balance, 1e-9)
	require.True(t, referredAfter.HasSuccessfulRecharge)

	commissions, err := client.ReferralCommission.Query().
		Where(
			referralcommission.ReferredUserIDEQ(referred.ID),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
		).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commissions, 2)

	firstCount := 0
	recurringCount := 0
	firstAmount := 0.0
	recurringAmount := 0.0
	for _, item := range commissions {
		switch item.CommissionType {
		case ReferralCommissionTypeFirst:
			firstCount++
			firstAmount = roundMoney(firstAmount + item.CommissionAmount)
		case ReferralCommissionTypeRecurring:
			recurringCount++
			recurringAmount = roundMoney(recurringAmount + item.CommissionAmount)
		}
	}
	require.Equal(t, 1, firstCount)
	require.Equal(t, 1, recurringCount)
	require.InDelta(t, 5.0, firstAmount, 1e-9)
	require.InDelta(t, 2.0, recurringAmount, 1e-9)
}

func TestReferralService_RefundRechargeOrder_ConcurrentCallsReverseCommissionOnce(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "false",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "0.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "promoter-refund@test.com", 0)
	referred, err := client.User.Create().
		SetEmail("referred-refund@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.RechargeOrder.Create().
		SetUserID(referred.ID).
		SetOrderNo("RC-CONCURRENT-REVERSE-COMMISSION-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(RechargeOrderStatusPending).
		Save(ctx)
	require.NoError(t, err)

	_, _, err = svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-CONCURRENT-REVERSE-COMMISSION-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
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
				_, _, err := svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
					OrderNo: "RC-CONCURRENT-REVERSE-COMMISSION-001",
					Channel: "manual",
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

	referredAfter, err := client.User.Get(ctx, referred.ID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, referredAfter.Balance, 1e-9)

	promoterAfter, err := client.User.Get(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, promoterAfter.Balance, 1e-9)

	commissions, err := client.ReferralCommission.Query().
		Where(referralcommission.PromoterUserIDEQ(promoterID)).
		All(ctx)
	require.NoError(t, err)
	require.Len(t, commissions, 1)
	require.Equal(t, ReferralCommissionStatusReversed, commissions[0].Status)
	require.Equal(t, ReferralCommissionTypeFirst, commissions[0].CommissionType)
	require.InDelta(t, 5.0, commissions[0].CommissionAmount, 1e-9)
}

func TestReferralService_RecordPaidRecharge_RefundDoesNotResetFirstRechargeIdentity(t *testing.T) {
	svc, client := newReferralServiceSQLiteWithSettings(t, map[string]string{
		SettingKeyAffiliateEnabled:               "true",
		SettingKeyFirstCommissionEnabled:         "true",
		SettingKeyRecurringCommissionEnabled:     "true",
		SettingKeyDefaultFirstCommissionRate:     "50.0000",
		SettingKeyDefaultRecurringCommissionRate: "20.0000",
	})
	ctx := context.Background()

	promoterID := mustCreateReferralTestUser(t, ctx, client, "first-identity-promoter@test.com", 0)
	_, err := client.User.UpdateOneID(promoterID).SetRecurringCommissionEnabled(true).Save(ctx)
	require.NoError(t, err)
	referred, err := client.User.Create().
		SetEmail("first-identity-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(RoleUser).
		SetStatus(StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	_, firstCommissions, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-FIRST-IDENTITY-001",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, firstCommissions, 1)
	require.Equal(t, ReferralCommissionTypeFirst, firstCommissions[0].CommissionType)

	_, _, err = svc.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
		OrderNo: "RC-FIRST-IDENTITY-001",
		Channel: "manual",
	})
	require.NoError(t, err)

	_, secondCommissions, err := svc.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
		UserID:         referred.ID,
		OrderNo:        "RC-FIRST-IDENTITY-002",
		Channel:        "manual",
		Amount:         10,
		CreditedAmount: 10,
		Currency:       "CNY",
	})
	require.NoError(t, err)
	require.Len(t, secondCommissions, 1)
	require.Equal(t, ReferralCommissionTypeRecurring, secondCommissions[0].CommissionType)
	require.InDelta(t, 2.0, secondCommissions[0].CommissionAmount, 1e-9)

	referredAfter, err := client.User.Get(ctx, referred.ID)
	require.NoError(t, err)
	require.True(t, referredAfter.HasSuccessfulRecharge)
}

func runWithSQLiteWriteRetry(fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < 8; attempt++ {
		lastErr = fn()
		if !isSQLiteLockError(lastErr) {
			return lastErr
		}
		time.Sleep(time.Duration(attempt+1) * 10 * time.Millisecond)
	}
	return lastErr
}

func isSQLiteLockError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") ||
		strings.Contains(msg, "database table is locked") ||
		strings.Contains(msg, "deadlocked")
}
