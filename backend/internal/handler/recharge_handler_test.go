//go:build unit

package handler

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/enttest"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "modernc.org/sqlite"
)

type rechargeSettingRepoStub struct {
	values map[string]string
}

func (s *rechargeSettingRepoStub) Get(ctx context.Context, key string) (*service.Setting, error) {
	if value, ok := s.values[key]; ok {
		return &service.Setting{Key: key, Value: value}, nil
	}
	return nil, service.ErrSettingNotFound
}

func (s *rechargeSettingRepoStub) GetValue(ctx context.Context, key string) (string, error) {
	if value, ok := s.values[key]; ok {
		return value, nil
	}
	return "", service.ErrSettingNotFound
}

func (s *rechargeSettingRepoStub) Set(ctx context.Context, key, value string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	s.values[key] = value
	return nil
}

func (s *rechargeSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := s.values[key]; ok {
			out[key] = value
		}
	}
	return out, nil
}

func (s *rechargeSettingRepoStub) SetMultiple(ctx context.Context, settings map[string]string) error {
	if s.values == nil {
		s.values = make(map[string]string)
	}
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *rechargeSettingRepoStub) GetAll(ctx context.Context) (map[string]string, error) {
	out := make(map[string]string, len(s.values))
	for key, value := range s.values {
		out[key] = value
	}
	return out, nil
}

func (s *rechargeSettingRepoStub) Delete(ctx context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func newRechargeHandlerTestClient(t *testing.T) *dbent.Client {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db := entsql.OpenDB(dialect.SQLite, mustOpenSQLite(t, dsn))
	client := enttest.NewClient(t, enttest.WithOptions(dbent.Driver(db)))
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func mustOpenSQLite(t *testing.T, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)
	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	require.NoError(t, err)
	return db
}

func mustCreateRechargeTestUser(t *testing.T, ctx context.Context, client *dbent.Client, email string) int64 {
	t.Helper()
	user, err := client.User.Create().
		SetEmail(email).
		SetPasswordHash("test-password-hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetBalance(0).
		SetConcurrency(5).
		Save(ctx)
	require.NoError(t, err)
	return user.ID
}

func newRechargeHandlerForTests(t *testing.T, queryResponder http.HandlerFunc) (*RechargeHandler, *dbent.Client, *service.ReferralService, *service.SettingService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	client := newRechargeHandlerTestClient(t)
	queryServer := httptest.NewServer(queryResponder)
	t.Cleanup(queryServer.Close)

	repo := &rechargeSettingRepoStub{
		values: map[string]string{
			service.SettingKeyXunhuPayEnabled:     "true",
			service.SettingKeyXunhuPayBaseURL:     queryServer.URL,
			service.SettingKeyXunhuPayAppID:       "test-app",
			service.SettingKeyXunhuPayAppSecret:   "test-secret",
			service.SettingKeyXunhuPayNotifyURL:   "https://merchant.example.com/api/v1/payments/webhook/xunhupay",
			service.SettingKeyBalanceRechargeRatio: "100",
		},
	}
	settingService := service.NewSettingService(repo, &config.Config{})
	referralService := service.NewReferralService(client, settingService, nil, nil, nil)
	xunhuPayService := service.NewXunhuPayService(settingService)

	return NewRechargeHandler(referralService, settingService, xunhuPayService), client, referralService, settingService
}

func makeAuthenticatedContext(method, path string, body *strings.Reader, userID int64) (*gin.Context, *httptest.ResponseRecorder) {
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	if body != nil {
		c.Request = httptest.NewRequest(method, path, body)
	} else {
		c.Request = httptest.NewRequest(method, path, nil)
	}
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: userID, Concurrency: 5})
	return c, rec
}

func signXunhuPayload(values map[string]string, secret string) string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if strings.EqualFold(key, "hash") || strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values[key])
	}
	sum := md5.Sum([]byte(strings.Join(parts, "&") + secret))
	return hex.EncodeToString(sum[:])
}

func makeXunhuQueryResponse(orderNo string, amount float64) string {
	payload := map[string]string{
		"errcode":         "0",
		"errmsg":          "ok",
		"out_trade_order": orderNo,
		"open_order_id":   "open-order-1",
		"transaction_id":  "tx-1",
		"status":          "OD",
		"total_fee":       strconv.FormatFloat(amount, 'f', 2, 64),
	}
	payload["hash"] = signXunhuPayload(payload, "test-secret")

	response := map[string]any{
		"errcode": 0,
		"errmsg":  "ok",
		"data": map[string]any{
			"out_trade_order": payload["out_trade_order"],
			"open_order_id":   payload["open_order_id"],
			"transaction_id":  payload["transaction_id"],
			"status":          payload["status"],
			"total_fee":       payload["total_fee"],
		},
		"hash": payload["hash"],
	}
	body, _ := json.Marshal(response)
	return string(body)
}

func makeXunhuCreateResponse(errCode int, message string) string {
	payload := map[string]string{
		"errcode": strconv.Itoa(errCode),
		"errmsg":  message,
	}
	if errCode == 0 {
		payload["open_order_id"] = "open-order-1"
		payload["url"] = "https://pay.example.com/pay/open-order-1"
		payload["url_qrcode"] = "https://pay.example.com/qr/open-order-1.png"
	}
	payload["hash"] = signXunhuPayload(payload, "test-secret")

	response := map[string]any{
		"errcode": errCode,
		"errmsg":  message,
		"hash":    payload["hash"],
	}
	if errCode == 0 {
		response["open_order_id"] = payload["open_order_id"]
		response["url"] = payload["url"]
		response["url_qrcode"] = payload["url_qrcode"]
	}
	body, _ := json.Marshal(response)
	return string(body)
}

func decodeRechargeCreateEnvelope(t *testing.T, body []byte) CreateRechargeOrderResponse {
	t.Helper()
	var envelope struct {
		Code int                       `json:"code"`
		Data CreateRechargeOrderResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	require.Equal(t, 0, envelope.Code)
	return envelope.Data
}

func makeXunhuWebhookForm(orderNo string, amount float64) url.Values {
	form := url.Values{}
	form.Set("appid", "test-app")
	form.Set("trade_order_id", orderNo)
	form.Set("open_order_id", "open-order-1")
	form.Set("transaction_id", "tx-1")
	form.Set("order_title", "Account Recharge")
	form.Set("status", "OD")
	form.Set("total_fee", strconv.FormatFloat(amount, 'f', 2, 64))
	form.Set("time", strconv.FormatInt(time.Now().Unix(), 10))

	payload := map[string]string{
		"appid":          form.Get("appid"),
		"trade_order_id": form.Get("trade_order_id"),
		"open_order_id":  form.Get("open_order_id"),
		"transaction_id": form.Get("transaction_id"),
		"order_title":    form.Get("order_title"),
		"status":         form.Get("status"),
		"total_fee":      form.Get("total_fee"),
		"time":           form.Get("time"),
	}
	form.Set("hash", signXunhuPayload(payload, "test-secret"))
	return form
}

func TestRechargeHandlerReconcileOrderUsesOrderCreditedAmountSnapshot(t *testing.T) {
	handler, client, referralService, _ := newRechargeHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/query.html", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeXunhuQueryResponse("RC-RECONCILE-PLACEHOLDER", 10)))
	})

	ctx := context.Background()
	userID := mustCreateRechargeTestUser(t, ctx, client, "reconcile-snapshot@test.com")
	order, err := referralService.CreateRechargeOrder(ctx, &service.CreateRechargeOrderInput{
		UserID:   userID,
		Amount:   10,
		Channel:  service.XunhuPayChannel,
		Currency: "CNY",
		Source:   service.RechargeOrderSourceBalance,
	})
	require.NoError(t, err)
	require.InDelta(t, 1000.0, order.CreditedAmount, 1e-9)

	queryServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/query.html", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeXunhuQueryResponse(order.OrderNo, 10)))
	}))
	defer queryServer.Close()

	settingRepo := &rechargeSettingRepoStub{
		values: map[string]string{
			service.SettingKeyXunhuPayEnabled:      "true",
			service.SettingKeyXunhuPayBaseURL:      queryServer.URL,
			service.SettingKeyXunhuPayAppID:        "test-app",
			service.SettingKeyXunhuPayAppSecret:    "test-secret",
			service.SettingKeyBalanceRechargeRatio: "100",
		},
	}
	settingService := service.NewSettingService(settingRepo, &config.Config{})
	referralService = service.NewReferralService(client, settingService, nil, nil, nil)
	handler = NewRechargeHandler(referralService, settingService, service.NewXunhuPayService(settingService))

	c, rec := makeAuthenticatedContext(http.MethodPost, "/api/v1/recharges/orders/"+order.OrderNo+"/reconcile", nil, userID)
	c.Params = gin.Params{{Key: "orderNo", Value: order.OrderNo}}

	handler.ReconcileOrder(c)

	require.Equal(t, http.StatusOK, rec.Code)
	user, err := client.User.Query().Where(dbuser.IDEQ(userID)).Only(ctx)
	require.NoError(t, err)
	require.InDelta(t, 1000.0, user.Balance, 1e-9)

	orderAfter, err := client.RechargeOrder.Query().Where(rechargeorder.OrderNoEQ(order.OrderNo)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, service.RechargeOrderStatusPaid, orderAfter.Status)
	require.InDelta(t, 1000.0, orderAfter.CreditedAmount, 1e-9)
}

func TestRechargeHandlerXunhuWebhookUsesOrderCreditedAmountSnapshot(t *testing.T) {
	handler, client, referralService, _ := newRechargeHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	ctx := context.Background()
	userID := mustCreateRechargeTestUser(t, ctx, client, "webhook-snapshot@test.com")
	order, err := referralService.CreateRechargeOrder(ctx, &service.CreateRechargeOrderInput{
		UserID:   userID,
		Amount:   10,
		Channel:  service.XunhuPayChannel,
		Currency: "CNY",
		Source:   service.RechargeOrderSourceBalance,
	})
	require.NoError(t, err)
	require.InDelta(t, 1000.0, order.CreditedAmount, 1e-9)

	form := makeXunhuWebhookForm(order.OrderNo, 10)
	body := strings.NewReader(form.Encode())
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/payments/webhook/xunhupay", body)
	c.Request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	handler.handleXunhuPayWebhook(c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "success", rec.Body.String())

	user, err := client.User.Query().Where(dbuser.IDEQ(userID)).Only(ctx)
	require.NoError(t, err)
	require.InDelta(t, 1000.0, user.Balance, 1e-9)

	orderAfter, err := client.RechargeOrder.Query().Where(rechargeorder.OrderNoEQ(order.OrderNo)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, service.RechargeOrderStatusPaid, orderAfter.Status)
	require.InDelta(t, 1000.0, orderAfter.CreditedAmount, 1e-9)
}

func TestRechargeHandlerCreateOrderIsIdempotent(t *testing.T) {
	var createCalls atomic.Int32
	handler, client, _, _ := newRechargeHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
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
	userID := mustCreateRechargeTestUser(t, ctx, client, "recharge-idempotent@test.com")

	router := gin.New()
	router.Use(withUserSubject(userID))
	router.POST("/recharges/orders", handler.CreateOrder)

	requestBody := `{"amount":10,"channel":"xunhupay","currency":"CNY","title":"Account Recharge"}`

	firstReq := httptest.NewRequest(http.MethodPost, "/recharges/orders", strings.NewReader(requestBody))
	firstReq.Header.Set("Content-Type", "application/json")
	firstReq.Header.Set("Idempotency-Key", "recharge-create-1")
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	secondReq := httptest.NewRequest(http.MethodPost, "/recharges/orders", strings.NewReader(requestBody))
	secondReq.Header.Set("Content-Type", "application/json")
	secondReq.Header.Set("Idempotency-Key", "recharge-create-1")
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Equal(t, http.StatusOK, secondRec.Code)
	require.Equal(t, "true", secondRec.Header().Get("X-Idempotency-Replayed"))

	first := decodeRechargeCreateEnvelope(t, firstRec.Body.Bytes())
	second := decodeRechargeCreateEnvelope(t, secondRec.Body.Bytes())
	require.Equal(t, first.Order.OrderNo, second.Order.OrderNo)
	require.Equal(t, int32(1), createCalls.Load())

	orders, err := client.RechargeOrder.Query().Where(rechargeorder.UserIDEQ(userID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, service.RechargeOrderStatusPending, orders[0].Status)
}

func TestRechargeHandlerCreateOrderMarksFailedWhenPaymentInitializationFails(t *testing.T) {
	handler, client, _, _ := newRechargeHandlerForTests(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/payment/do.html", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(makeXunhuCreateResponse(1001, "provider unavailable")))
	})

	ctx := context.Background()
	userID := mustCreateRechargeTestUser(t, ctx, client, "recharge-create-failed@test.com")

	router := gin.New()
	router.Use(withUserSubject(userID))
	router.POST("/recharges/orders", handler.CreateOrder)

	req := httptest.NewRequest(http.MethodPost, "/recharges/orders", strings.NewReader(`{"amount":10,"channel":"xunhupay","currency":"CNY","title":"Account Recharge"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "recharge-failure-1")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)

	orders, err := client.RechargeOrder.Query().Where(rechargeorder.UserIDEQ(userID)).All(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	require.Equal(t, service.RechargeOrderStatusFailed, orders[0].Status)
}
