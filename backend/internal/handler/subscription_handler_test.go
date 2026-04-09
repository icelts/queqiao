//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type purchaseGroupRepoStub struct {
	group *service.Group
}

func (purchaseGroupRepoStub) Create(context.Context, *service.Group) error { panic("unexpected Create call") }
func (s purchaseGroupRepoStub) GetByID(context.Context, int64) (*service.Group, error) {
	if s.group == nil {
		return nil, service.ErrGroupNotFound
	}
	return s.group, nil
}
func (purchaseGroupRepoStub) GetByIDLite(context.Context, int64) (*service.Group, error) {
	panic("unexpected GetByIDLite call")
}
func (purchaseGroupRepoStub) Update(context.Context, *service.Group) error { panic("unexpected Update call") }
func (purchaseGroupRepoStub) Delete(context.Context, int64) error { panic("unexpected Delete call") }
func (purchaseGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected DeleteCascade call")
}
func (purchaseGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (purchaseGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]service.Group, *pagination.PaginationResult, error) {
	panic("unexpected ListWithFilters call")
}
func (purchaseGroupRepoStub) ListActive(context.Context) ([]service.Group, error) { panic("unexpected ListActive call") }
func (purchaseGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]service.Group, error) {
	panic("unexpected ListActiveByPlatform call")
}
func (purchaseGroupRepoStub) ExistsByName(context.Context, string) (bool, error) { panic("unexpected ExistsByName call") }
func (purchaseGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected GetAccountCount call")
}
func (purchaseGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected DeleteAccountGroupsByGroupID call")
}
func (purchaseGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected GetAccountIDsByGroupIDs call")
}
func (purchaseGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected BindAccountsToGroup call")
}
func (purchaseGroupRepoStub) UpdateSortOrders(context.Context, []service.GroupSortOrderUpdate) error {
	panic("unexpected UpdateSortOrders call")
}

type purchaseUserSubRepoStub struct{}

func (purchaseUserSubRepoStub) Create(context.Context, *service.UserSubscription) error {
	panic("unexpected Create call")
}
func (purchaseUserSubRepoStub) GetByID(context.Context, int64) (*service.UserSubscription, error) {
	panic("unexpected GetByID call")
}
func (purchaseUserSubRepoStub) GetByUserIDAndGroupID(context.Context, int64, int64) (*service.UserSubscription, error) {
	return nil, service.ErrSubscriptionNotFound
}
func (purchaseUserSubRepoStub) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*service.UserSubscription, error) {
	panic("unexpected GetActiveByUserIDAndGroupID call")
}
func (purchaseUserSubRepoStub) Update(context.Context, *service.UserSubscription) error {
	panic("unexpected Update call")
}
func (purchaseUserSubRepoStub) Delete(context.Context, int64) error { panic("unexpected Delete call") }
func (purchaseUserSubRepoStub) ListByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	panic("unexpected ListByUserID call")
}
func (purchaseUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]service.UserSubscription, error) {
	panic("unexpected ListActiveByUserID call")
}
func (purchaseUserSubRepoStub) ListByGroupID(context.Context, int64, pagination.PaginationParams) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected ListByGroupID call")
}
func (purchaseUserSubRepoStub) List(context.Context, pagination.PaginationParams, *int64, *int64, string, string, string, string) ([]service.UserSubscription, *pagination.PaginationResult, error) {
	panic("unexpected List call")
}
func (purchaseUserSubRepoStub) ExistsByUserIDAndGroupID(context.Context, int64, int64) (bool, error) {
	panic("unexpected ExistsByUserIDAndGroupID call")
}
func (purchaseUserSubRepoStub) ExtendExpiry(context.Context, int64, time.Time) error {
	panic("unexpected ExtendExpiry call")
}
func (purchaseUserSubRepoStub) UpdateStatus(context.Context, int64, string) error {
	panic("unexpected UpdateStatus call")
}
func (purchaseUserSubRepoStub) UpdateNotes(context.Context, int64, string) error {
	panic("unexpected UpdateNotes call")
}
func (purchaseUserSubRepoStub) ActivateWindows(context.Context, int64, time.Time) error {
	panic("unexpected ActivateWindows call")
}
func (purchaseUserSubRepoStub) ResetDailyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetDailyUsage call")
}
func (purchaseUserSubRepoStub) ResetWeeklyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetWeeklyUsage call")
}
func (purchaseUserSubRepoStub) ResetMonthlyUsage(context.Context, int64, time.Time) error {
	panic("unexpected ResetMonthlyUsage call")
}
func (purchaseUserSubRepoStub) IncrementUsage(context.Context, int64, float64) error {
	panic("unexpected IncrementUsage call")
}
func (purchaseUserSubRepoStub) BatchUpdateExpiredStatus(context.Context) (int64, error) {
	panic("unexpected BatchUpdateExpiredStatus call")
}

func decodeSubscriptionCreateEnvelope(t *testing.T, body []byte) SubscriptionPurchaseOrderResponse {
	t.Helper()
	var envelope struct {
		Code int                             `json:"code"`
		Data SubscriptionPurchaseOrderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func newSubscriptionHandlerForTests(t *testing.T, createResponder http.HandlerFunc) (*SubscriptionHandler, *dbent.Client) {
	t.Helper()
	client := newRechargeHandlerTestClient(t)
	createServer := httptest.NewServer(createResponder)
	t.Cleanup(createServer.Close)

	settingService := service.NewSettingService(&rechargeSettingRepoStub{
		values: map[string]string{
			service.SettingKeyXunhuPayEnabled:      "true",
			service.SettingKeyXunhuPayBaseURL:      createServer.URL,
			service.SettingKeyXunhuPayAppID:        "test-app",
			service.SettingKeyXunhuPayAppSecret:    "test-secret",
			service.SettingKeyXunhuPayNotifyURL:    "https://merchant.example.com/api/v1/payments/webhook/xunhupay",
			service.SettingKeyBalanceRechargeRatio: "100",
		},
	}, &config.Config{})

	referralService := service.NewReferralService(client, settingService, nil, nil, nil)
	price := 68.0
	subscriptionService := service.NewSubscriptionService(
		purchaseGroupRepoStub{group: &service.Group{
			ID:                  7,
			Name:                "Pro",
			Platform:            service.PlatformAnthropic,
			Status:              service.StatusActive,
			SubscriptionType:    service.SubscriptionTypeSubscription,
			PurchaseEnabled:     true,
			PurchasePrice:       &price,
			DefaultValidityDays: 30,
		}},
		purchaseUserSubRepoStub{},
		nil,
		nil,
		nil,
	)

	return NewSubscriptionHandler(subscriptionService, referralService, service.NewXunhuPayService(settingService)), client
}

func TestSubscriptionHandlerCreatePurchaseOrderIsIdempotent(t *testing.T) {
	var createCalls atomic.Int32
	handler, client := newSubscriptionHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/do.html", r.URL.Path)
		createCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeXunhuCreateResponse(0, "ok")))
	})

	repo := newUserMemoryIdempotencyRepoStub()
	cfg := service.DefaultIdempotencyConfig()
	cfg.ObserveOnly = false
	service.SetDefaultIdempotencyCoordinator(service.NewIdempotencyCoordinator(repo, cfg))
	t.Cleanup(func() {
		service.SetDefaultIdempotencyCoordinator(nil)
	})

	ctx := context.Background()
	userID := mustCreateRechargeTestUser(t, ctx, client, "subscription-idempotent@test.com")

	router := gin.New()
	router.Use(withUserSubject(userID))
	router.POST("/subscriptions/purchase-orders", handler.CreatePurchaseOrder)

	requestBody := `{"group_id":7}`

	firstReq := httptest.NewRequest(http.MethodPost, "/subscriptions/purchase-orders", strings.NewReader(requestBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "subscription-create-1")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	secondReq := httptest.NewRequest(http.MethodPost, "/subscriptions/purchase-orders", strings.NewReader(requestBody))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "subscription-create-1")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Equal(t, http.StatusOK, secondRec.Code)
	require.Equal(t, "true", secondRec.Header().Get("X-Idempotency-Replayed"))

	first := decodeSubscriptionCreateEnvelope(t, firstRec.Body.Bytes())
	second := decodeSubscriptionCreateEnvelope(t, secondRec.Body.Bytes())
	require.Equal(t, first.Order.OrderNo, second.Order.OrderNo)
	require.Equal(t, int32(1), createCalls.Load())

	orders, err := client.RechargeOrder.Query().Where(rechargeorder.UserIDEQ(userID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, service.RechargeOrderSourceSubscriptionPurchase, orders[0].Source)
	require.Equal(t, service.RechargeOrderStatusPending, orders[0].Status)
}

func TestSubscriptionHandlerCreatePurchaseOrderMarksFailedWhenPaymentInitializationFails(t *testing.T) {
	handler, client := newSubscriptionHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/do.html", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeXunhuCreateResponse(1001, "provider unavailable")))
	})

	ctx := context.Background()
	userID := mustCreateRechargeTestUser(t, ctx, client, "subscription-create-failed@test.com")

	router := gin.New()
	router.Use(withUserSubject(userID))
	router.POST("/subscriptions/purchase-orders", handler.CreatePurchaseOrder)

	req := httptest.NewRequest(http.MethodPost, "/subscriptions/purchase-orders", strings.NewReader(`{"group_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "subscription-failure-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	orders, err := client.RechargeOrder.Query().Where(rechargeorder.UserIDEQ(userID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, service.RechargeOrderSourceSubscriptionPurchase, orders[0].Source)
	require.Equal(t, service.RechargeOrderStatusFailed, orders[0].Status)
}
