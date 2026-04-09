package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	XunhuPayChannel            = "xunhupay"
	defaultXunhuPayBaseURL     = "https://api.xunhupay.com"
	defaultXunhuPayPlugins     = "sub2api"
	xunhuPayCreateEndpointPath = "/payment/do.html"
	xunhuPayQueryEndpointPath  = "/payment/query.html"
	xunhuPayVersion            = "1.1"
	xunhuPayStatusPaid         = "OD"
	xunhuPayStatusRefunded     = "CD"
	xunhuPayStatusRefunding    = "RD"
	xunhuPayStatusRefundFailed = "UD"
	xunhuPayStatusPending      = "WP"
)

var (
	ErrXunhuPayDisabled         = infraerrors.Forbidden("XUNHUPAY_DISABLED", "xunhupay is disabled")
	ErrXunhuPayNotConfigured    = infraerrors.BadRequest("XUNHUPAY_NOT_CONFIGURED", "xunhupay is not configured")
	ErrXunhuPayInvalidSignature = infraerrors.Unauthorized("XUNHUPAY_INVALID_SIGNATURE", "invalid xunhupay signature")
	ErrXunhuPayCreateFailed     = infraerrors.BadRequest("XUNHUPAY_CREATE_FAILED", "failed to create xunhupay payment")
	ErrXunhuPayQueryFailed      = infraerrors.BadRequest("XUNHUPAY_QUERY_FAILED", "failed to query xunhupay payment")
)

type XunhuPaySettings struct {
	Enabled     bool
	BaseURL     string
	AppID       string
	AppSecret   string
	NotifyURL   string
	ReturnURL   string
	CallbackURL string
	Plugins     string
}

type XunhuCreatePaymentInput struct {
	OrderNo string
	Amount  float64
	Title   string
	Attach  string
}

type XunhuCreatePaymentResult struct {
	Provider    string         `json:"provider"`
	OpenOrderID string         `json:"open_order_id,omitempty"`
	PaymentURL  string         `json:"payment_url,omitempty"`
	QRCodeURL   string         `json:"qrcode_url,omitempty"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	ResponseRaw map[string]any `json:"response_raw,omitempty"`
}

type XunhuQueryPaymentInput struct {
	OrderNo     string
	OpenOrderID string
}

type XunhuQueryPaymentResult struct {
	Provider      string         `json:"provider"`
	Status        string         `json:"status"`
	OrderNo       string         `json:"order_no,omitempty"`
	OpenOrderID   string         `json:"open_order_id,omitempty"`
	TransactionID string         `json:"transaction_id,omitempty"`
	TotalFee      float64        `json:"total_fee,omitempty"`
	ResponseRaw   map[string]any `json:"response_raw,omitempty"`
}

type XunhuWebhookNotification struct {
	TradeOrderID   string
	OpenOrderID    string
	TransactionID  string
	OrderTitle     string
	Status         string
	TotalFee       float64
	Plugins        string
	Attach         string
	AppID          string
	OccurredAt     *time.Time
	IdempotencyKey string
	Raw            string
}

type XunhuPayService struct {
	settingService *SettingService
	httpClient     *http.Client
}

func NewXunhuPayService(settingService *SettingService) *XunhuPayService {
	return &XunhuPayService{
		settingService: settingService,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (s *XunhuPayService) CreatePayment(ctx context.Context, input *XunhuCreatePaymentInput) (*XunhuCreatePaymentResult, error) {
	if input == nil || strings.TrimSpace(input.OrderNo) == "" || input.Amount <= 0 {
		return nil, ErrRechargeOrderInvalid
	}

	cfg, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateForCreate(); err != nil {
		return nil, err
	}

	nonceStr, err := randomHexString(8)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XUNHUPAY_NONCE_FAILED", "generate nonce failed")
	}

	title := sanitizeXunhuTitle(input.Title)
	if title == "" {
		title = "Account Recharge"
	}

	payload := map[string]string{
		"version":        xunhuPayVersion,
		"appid":          cfg.AppID,
		"trade_order_id": strings.TrimSpace(input.OrderNo),
		"total_fee":      strconv.FormatFloat(roundMoney(input.Amount), 'f', 2, 64),
		"title":          title,
		"time":           strconv.FormatInt(time.Now().Unix(), 10),
		"notify_url":     cfg.NotifyURL,
		"nonce_str":      nonceStr,
	}
	if cfg.ReturnURL != "" {
		payload["return_url"] = cfg.ReturnURL
	}
	if cfg.CallbackURL != "" {
		payload["callback_url"] = cfg.CallbackURL
	}
	if cfg.Plugins != "" {
		payload["plugins"] = cfg.Plugins
	}
	if attach := strings.TrimSpace(input.Attach); attach != "" {
		payload["attach"] = attach
	}
	payload["hash"] = generateXunhuHash(payload, cfg.AppSecret)

	values := url.Values{}
	for key, value := range payload {
		values.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+xunhuPayCreateEndpointPath, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XUNHUPAY_REQUEST_BUILD_FAILED", "build xunhupay request failed")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_REQUEST_FAILED", "request xunhupay failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_RESPONSE_READ_FAILED", "read xunhupay response failed")
	}

	var responseMap map[string]any
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_RESPONSE_INVALID", "invalid xunhupay response")
	}
	if !verifyXunhuSignature(responseMap, cfg.AppSecret) {
		return nil, ErrXunhuPayInvalidSignature
	}

	errCode := anyToInt(responseMap["errcode"])
	if resp.StatusCode >= http.StatusBadRequest || errCode != 0 {
		message := strings.TrimSpace(anyToString(responseMap["errmsg"]))
		if message == "" {
			message = "failed to create xunhupay payment"
		}
		return nil, ErrXunhuPayCreateFailed.WithMetadata(map[string]string{
			"provider_code": strconv.Itoa(errCode),
			"provider_msg":  message,
		})
	}

	expiresAt := time.Now().Add(5 * time.Minute).UTC()
	return &XunhuCreatePaymentResult{
		Provider:    XunhuPayChannel,
		OpenOrderID: firstNonBlank(anyToString(responseMap["open_order_id"]), anyToString(responseMap["openid"])),
		PaymentURL:  strings.TrimSpace(anyToString(responseMap["url"])),
		QRCodeURL:   strings.TrimSpace(anyToString(responseMap["url_qrcode"])),
		ExpiresAt:   &expiresAt,
		ResponseRaw: responseMap,
	}, nil
}

func (s *XunhuPayService) QueryPayment(ctx context.Context, input *XunhuQueryPaymentInput) (*XunhuQueryPaymentResult, error) {
	if input == nil {
		return nil, ErrRechargeOrderInvalid
	}

	orderNo := strings.TrimSpace(input.OrderNo)
	openOrderID := strings.TrimSpace(input.OpenOrderID)
	if orderNo == "" && openOrderID == "" {
		return nil, ErrRechargeOrderInvalid
	}

	cfg, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateForQuery(); err != nil {
		return nil, err
	}

	nonceStr, err := randomHexString(8)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XUNHUPAY_NONCE_FAILED", "generate nonce failed")
	}

	payload := map[string]string{
		"appid":     cfg.AppID,
		"time":      strconv.FormatInt(time.Now().Unix(), 10),
		"nonce_str": nonceStr,
	}
	if openOrderID != "" {
		payload["open_order_id"] = openOrderID
	} else if orderNo != "" {
		payload["out_trade_order"] = orderNo
	}
	payload["hash"] = generateXunhuHash(payload, cfg.AppSecret)

	values := url.Values{}
	for key, value := range payload {
		values.Set(key, value)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.BaseURL, "/")+xunhuPayQueryEndpointPath, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "XUNHUPAY_REQUEST_BUILD_FAILED", "build xunhupay request failed")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_REQUEST_FAILED", "request xunhupay failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_RESPONSE_READ_FAILED", "read xunhupay response failed")
	}

	var responseMap map[string]any
	if err := json.Unmarshal(body, &responseMap); err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "XUNHUPAY_RESPONSE_INVALID", "invalid xunhupay response")
	}
	if !verifyXunhuQuerySignature(responseMap, cfg.AppSecret) {
		return nil, ErrXunhuPayInvalidSignature
	}

	errCode := anyToInt(responseMap["errcode"])
	if resp.StatusCode >= http.StatusBadRequest || errCode != 0 {
		message := strings.TrimSpace(anyToString(responseMap["errmsg"]))
		if message == "" {
			message = "failed to query xunhupay payment"
		}
		return nil, ErrXunhuPayQueryFailed.WithMetadata(map[string]string{
			"provider_code": strconv.Itoa(errCode),
			"provider_msg":  message,
		})
	}

	dataMap := castAnyMap(responseMap["data"])
	status := strings.ToUpper(strings.TrimSpace(firstNonBlank(
		anyToString(dataMap["status"]),
		anyToString(responseMap["status"]),
	)))
	totalFee := roundMoney(anyToFloat64(firstNonNil(dataMap["total_fee"], responseMap["total_fee"])))

	return &XunhuQueryPaymentResult{
		Provider: XunhuPayChannel,
		Status:   status,
		OrderNo: firstNonBlank(
			anyToString(dataMap["out_trade_order"]),
			anyToString(dataMap["trade_order_id"]),
			orderNo,
		),
		OpenOrderID: firstNonBlank(
			anyToString(dataMap["open_order_id"]),
			anyToString(responseMap["open_order_id"]),
			openOrderID,
		),
		TransactionID: firstNonBlank(
			anyToString(dataMap["transaction_id"]),
			anyToString(responseMap["transaction_id"]),
		),
		TotalFee:    totalFee,
		ResponseRaw: responseMap,
	}, nil
}

func (s *XunhuPayService) VerifyWebhook(ctx context.Context, values url.Values) (*XunhuWebhookNotification, error) {
	cfg, err := s.loadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if err := cfg.validateForWebhook(); err != nil {
		return nil, err
	}

	flat := flattenFormValues(values)
	if !verifyXunhuStringMapSignature(flat, cfg.AppSecret) {
		return nil, ErrXunhuPayInvalidSignature
	}

	appID := strings.TrimSpace(flat["appid"])
	if appID != "" && appID != cfg.AppID {
		return nil, ErrXunhuPayInvalidSignature
	}

	totalFee, _ := strconv.ParseFloat(strings.TrimSpace(flat["total_fee"]), 64)
	var occurredAt *time.Time
	if unixValue, err := strconv.ParseInt(strings.TrimSpace(flat["time"]), 10, 64); err == nil && unixValue > 0 {
		timestamp := time.Unix(unixValue, 0).UTC()
		occurredAt = &timestamp
	}

	status := strings.ToUpper(strings.TrimSpace(flat["status"]))
	idempotencyKey := firstNonBlank(
		strings.TrimSpace(flat["transaction_id"]),
		strings.TrimSpace(flat["open_order_id"]),
		strings.TrimSpace(flat["trade_order_id"])+":"+status+":"+strings.TrimSpace(flat["time"]),
	)

	return &XunhuWebhookNotification{
		TradeOrderID:   strings.TrimSpace(flat["trade_order_id"]),
		OpenOrderID:    strings.TrimSpace(flat["open_order_id"]),
		TransactionID:  strings.TrimSpace(flat["transaction_id"]),
		OrderTitle:     strings.TrimSpace(flat["order_title"]),
		Status:         status,
		TotalFee:       roundMoney(totalFee),
		Plugins:        strings.TrimSpace(flat["plugins"]),
		Attach:         strings.TrimSpace(flat["attach"]),
		AppID:          appID,
		OccurredAt:     occurredAt,
		IdempotencyKey: idempotencyKey,
		Raw:            values.Encode(),
	}, nil
}

func (s *XunhuPayService) loadSettings(ctx context.Context) (*XunhuPaySettings, error) {
	if s.settingService == nil {
		return nil, ErrXunhuPayNotConfigured
	}

	settings, err := s.settingService.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}

	return &XunhuPaySettings{
		Enabled:     settings.XunhuPayEnabled,
		BaseURL:     firstNonBlank(settings.XunhuPayBaseURL, defaultXunhuPayBaseURL),
		AppID:       strings.TrimSpace(settings.XunhuPayAppID),
		AppSecret:   strings.TrimSpace(settings.XunhuPayAppSecret),
		NotifyURL:   strings.TrimSpace(settings.XunhuPayNotifyURL),
		ReturnURL:   strings.TrimSpace(settings.XunhuPayReturnURL),
		CallbackURL: strings.TrimSpace(settings.XunhuPayCallbackURL),
		Plugins:     firstNonBlank(settings.XunhuPayPlugins, defaultXunhuPayPlugins),
	}, nil
}

func (c *XunhuPaySettings) validateForCreate() error {
	if c == nil || !c.Enabled {
		return ErrXunhuPayDisabled
	}
	if strings.TrimSpace(c.AppID) == "" || strings.TrimSpace(c.AppSecret) == "" || strings.TrimSpace(c.NotifyURL) == "" {
		return ErrXunhuPayNotConfigured
	}
	return nil
}

func (c *XunhuPaySettings) validateForWebhook() error {
	if c == nil || !c.Enabled {
		return ErrXunhuPayDisabled
	}
	if strings.TrimSpace(c.AppID) == "" || strings.TrimSpace(c.AppSecret) == "" {
		return ErrXunhuPayNotConfigured
	}
	return nil
}

func (c *XunhuPaySettings) validateForQuery() error {
	return c.validateForWebhook()
}

func sanitizeXunhuTitle(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "%", "")
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) > 42 {
		runes = runes[:42]
	}
	return string(runes)
}

func flattenFormValues(values url.Values) map[string]string {
	result := make(map[string]string, len(values))
	for key, items := range values {
		if len(items) == 0 {
			continue
		}
		result[key] = strings.TrimSpace(items[0])
	}
	return result
}

func verifyXunhuSignature(payload map[string]any, secret string) bool {
	received := strings.TrimSpace(anyToString(payload["hash"]))
	if received == "" {
		return false
	}
	expected := generateXunhuHash(anyMapToStringMap(payload), secret)
	return strings.EqualFold(received, expected)
}

func verifyXunhuStringMapSignature(payload map[string]string, secret string) bool {
	received := strings.TrimSpace(payload["hash"])
	if received == "" {
		return false
	}
	expected := generateXunhuHash(payload, secret)
	return strings.EqualFold(received, expected)
}

func verifyXunhuQuerySignature(payload map[string]any, secret string) bool {
	if verifyXunhuSignature(payload, secret) {
		return true
	}

	received := strings.TrimSpace(anyToString(payload["hash"]))
	if received == "" {
		return false
	}

	flattened := make(map[string]string, len(payload))
	for key, value := range payload {
		if strings.EqualFold(key, "hash") || strings.EqualFold(key, "data") {
			continue
		}
		flattened[key] = anyToString(value)
	}

	dataMap := castAnyMap(payload["data"])
	for key, value := range dataMap {
		flattened[key] = anyToString(value)
	}

	expected := generateXunhuHash(flattened, secret)
	return strings.EqualFold(received, expected)
}

func generateXunhuHash(payload map[string]string, secret string) string {
	keys := make([]string, 0, len(payload))
	for key, value := range payload {
		if strings.EqualFold(key, "hash") {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+payload[key])
	}

	sum := md5.Sum([]byte(strings.Join(parts, "&") + secret))
	return hex.EncodeToString(sum[:])
}

func anyMapToStringMap(payload map[string]any) map[string]string {
	result := make(map[string]string, len(payload))
	for key, value := range payload {
		result[key] = anyToString(value)
	}
	return result
}

func anyToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case int32:
		return strconv.FormatInt(int64(typed), 10)
	case uint:
		return strconv.FormatUint(uint64(typed), 10)
	case uint64:
		return strconv.FormatUint(typed, 10)
	case uint32:
		return strconv.FormatUint(uint64(typed), 10)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(value)
	}
}

func anyToInt(value any) int {
	switch typed := value.(type) {
	case nil:
		return 0
	case int:
		return typed
	case int64:
		return int(typed)
	case int32:
		return int(typed)
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return int(parsed)
		}
		if parsed, err := typed.Float64(); err == nil {
			return int(parsed)
		}
	case string:
		if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
			return parsed
		}
	}
	return 0
}

func anyToFloat64(value any) float64 {
	switch typed := value.(type) {
	case nil:
		return 0
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case int32:
		return float64(typed)
	case uint:
		return float64(typed)
	case uint64:
		return float64(typed)
	case uint32:
		return float64(typed)
	case json.Number:
		if parsed, err := typed.Float64(); err == nil {
			return parsed
		}
	case string:
		if parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return parsed
		}
	}
	return 0
}

func castAnyMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return typed
}

func firstNonNil(values ...any) any {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
