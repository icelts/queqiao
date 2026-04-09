//go:build unit

package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalallocation"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalrequest"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newReferralHandlerForTests(t *testing.T) (*ReferralHandler, *service.ReferralService, *dbent.Client, int64) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := newRechargeHandlerTestClient(t)
	settingService := service.NewSettingService(&rechargeSettingRepoStub{
		values: map[string]string{
			service.SettingKeyAffiliateEnabled:               "true",
			service.SettingKeyFirstCommissionEnabled:         "true",
			service.SettingKeyRecurringCommissionEnabled:     "false",
			service.SettingKeyDefaultFirstCommissionRate:     "50.0000",
			service.SettingKeyDefaultRecurringCommissionRate: "0.0000",
			service.SettingKeyAffiliateWithdrawEnabled:       "true",
			service.SettingKeyAffiliateWithdrawMinAmount:     "1.00000000",
			service.SettingKeyAffiliateWithdrawMinInvitees:   "0",
		},
	}, &config.Config{})
	referralService := service.NewReferralService(client, settingService, nil, nil, nil)

	ctx := context.Background()
	promoterID := mustCreateRechargeTestUser(t, ctx, client, "referral-handler-promoter@test.com")
	referred, err := client.User.Create().
		SetEmail("referral-handler-referred@test.com").
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetInviterID(promoterID).
		Save(ctx)
	require.NoError(t, err)

	paidAt := time.Now().Add(-40 * 24 * time.Hour)
	order, err := client.RechargeOrder.Create().
		SetUserID(referred.ID).
		SetOrderNo("RC-REFERRAL-HANDLER-001").
		SetChannel("manual").
		SetSource("payment").
		SetCurrency("CNY").
		SetAmount(10).
		SetCreditedAmount(10).
		SetStatus(service.RechargeOrderStatusPaid).
		SetPaidAt(paidAt).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.ReferralCommission.Create().
		SetPromoterUserID(promoterID).
		SetReferredUserID(referred.ID).
		SetRechargeOrderID(order.ID).
		SetCommissionType(service.ReferralCommissionTypeFirst).
		SetStatus(service.ReferralCommissionStatusRecorded).
		SetSourceAmount(10).
		SetRateSnapshot(50).
		SetCommissionAmount(5).
		SetCurrency("CNY").
		SetCreatedAt(paidAt).
		Save(ctx)
	require.NoError(t, err)
	_, err = client.User.UpdateOneID(referred.ID).SetHasSuccessfulRecharge(true).Save(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = client.Close()
	})
	return NewReferralHandler(referralService), referralService, client, promoterID
}

func decodeReferralWithdrawalEnvelope(t *testing.T, body []byte) referralWithdrawalRequestResponse {
	t.Helper()
	var envelope struct {
		Code int                               `json:"code"`
		Data referralWithdrawalRequestResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func TestReferralHandlerCreateWithdrawalRequestIsIdempotent(t *testing.T) {
	handler, referralService, client, promoterID := newReferralHandlerForTests(t)

	repo := newUserMemoryIdempotencyRepoStub()
	cfg := service.DefaultIdempotencyConfig()
	cfg.ObserveOnly = false
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, cfg))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	ctx := context.Background()
	summary, err := referralService.GetSummary(ctx, promoterID)
	require.NoError(t, err)
	require.InDelta(t, 5.0, summary.AvailableCommission, 1e-9)

	router := gin.New()
	router.Use(withUserSubject(promoterID))
	router.POST("/referral/withdrawals", handler.CreateWithdrawalRequest)

	body := `{"amount":2,"payment_method":"alipay","account_identifier":"alice@example","notes":"first"}`
	firstReq := httptest.NewRequest(http.MethodPost, "/referral/withdrawals", strings.NewReader(body))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "referral-withdrawal-create-1")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	secondReq := httptest.NewRequest(http.MethodPost, "/referral/withdrawals", strings.NewReader(body))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "referral-withdrawal-create-1")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Equal(t, http.StatusOK, secondRec.Code)
	require.Equal(t, "true", secondRec.Header().Get("X-Idempotency-Replayed"))

	first := decodeReferralWithdrawalEnvelope(t, firstRec.Body.Bytes())
	second := decodeReferralWithdrawalEnvelope(t, secondRec.Body.Bytes())
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, service.ReferralWithdrawalStatusPending, first.Status)

	requestCount, err := client.ReferralWithdrawalRequest.Query().
		Where(
			referralwithdrawalrequest.PromoterUserIDEQ(promoterID),
			referralwithdrawalrequest.StatusEQ(service.ReferralWithdrawalStatusPending),
		).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, requestCount)

	allocationCount, err := client.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.PromoterUserIDEQ(promoterID)).
		Count(ctx)
	require.NoError(t, err)
	require.Greater(t, allocationCount, 0)
}

func TestReferralHandlerCreateWithdrawalRequestRequiresIdempotencyKey(t *testing.T) {
	handler, _, client, promoterID := newReferralHandlerForTests(t)

	router := gin.New()
	router.Use(withUserSubject(promoterID))
	router.POST("/referral/withdrawals", handler.CreateWithdrawalRequest)

	req := httptest.NewRequest(http.MethodPost, "/referral/withdrawals", strings.NewReader(`{"amount":2,"payment_method":"alipay","account_identifier":"alice@example"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	ctx := context.Background()
	count, err := client.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.PromoterUserIDEQ(promoterID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	referredCount, err := client.User.Query().
		Where(dbuser.InviterIDEQ(promoterID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, referredCount)

	orderCount, err := client.RechargeOrder.Query().
		Where(rechargeorder.UserIDEQ(promoterID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, orderCount)
}

func TestReferralHandlerCreateWithdrawalRequestRejectsOversizedBody(t *testing.T) {
	handler, _, client, promoterID := newReferralHandlerForTests(t)

	router := gin.New()
	router.Use(withUserSubject(promoterID))
	router.POST("/referral/withdrawals", servermiddleware.RequestBodyLimit(64), handler.CreateWithdrawalRequest)

	payload := `{"amount":2,"payment_method":"alipay","account_identifier":"alice@example","notes":"` + strings.Repeat("a", 256) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/referral/withdrawals", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "oversized-referral-body")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	require.Contains(t, rec.Body.String(), buildBodyTooLargeMessage(64))

	ctx := context.Background()
	count, err := client.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.PromoterUserIDEQ(promoterID)).
		Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
