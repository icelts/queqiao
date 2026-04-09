package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type RechargeHandler struct {
	referralService *service.ReferralService
	settingService  *service.SettingService
	xunhuPayService *service.XunhuPayService
}

type CreateRechargeOrderRequest struct {
	Amount   float64 `json:"amount" binding:"required,gt=0"`
	Channel  string  `json:"channel"`
	Currency string  `json:"currency"`
	Title    string  `json:"title"`
	Notes    string  `json:"notes"`
}

type PaymentWebhookRequest struct {
	OrderNo                string  `json:"order_no"`
	ExternalOrderID        string  `json:"external_order_id"`
	Status                 string  `json:"status"`
	Amount                 float64 `json:"amount"`
	CreditedAmount         float64 `json:"credited_amount"`
	Currency               string  `json:"currency"`
	CallbackIdempotencyKey string  `json:"callback_idempotency_key"`
	Notes                  string  `json:"notes"`
}

type CreateRechargeOrderResponse struct {
	Order   *service.RechargeOrder            `json:"order"`
	Payment *service.XunhuCreatePaymentResult `json:"payment,omitempty"`
}

type ReconcileRechargeOrderResponse struct {
	Order       *service.RechargeOrder           `json:"order"`
	Payment     *service.XunhuQueryPaymentResult `json:"payment,omitempty"`
	Commissions []service.ReferralCommission     `json:"commissions,omitempty"`
}

const (
	paymentWebhookTimestampHeader = "x-webhook-timestamp"
	paymentWebhookSignatureHeader = "x-webhook-signature"
	paymentWebhookMaxSkew         = 5 * time.Minute
)

func NewRechargeHandler(referralService *service.ReferralService, settingService *service.SettingService, xunhuPayService *service.XunhuPayService) *RechargeHandler {
	return &RechargeHandler{
		referralService: referralService,
		settingService:  settingService,
		xunhuPayService: xunhuPayService,
	}
}

func (h *RechargeHandler) CreateOrder(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	if !requireIdempotencyKey(c) {
		return
	}

	var req CreateRechargeOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	channel := normalizeRechargeChannel(req.Channel)
	currency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if currency == "" {
		currency = "CNY"
	}
	payload := struct {
		Amount   float64 `json:"amount"`
		Channel  string  `json:"channel"`
		Currency string  `json:"currency"`
		Title    string  `json:"title"`
		Notes    string  `json:"notes"`
	}{
		Amount:   req.Amount,
		Channel:  channel,
		Currency: currency,
		Title:    strings.TrimSpace(req.Title),
		Notes:    strings.TrimSpace(req.Notes),
	}

	executeUserIdempotentJSON(
		c,
		"user.recharge.create_order",
		payload,
		service.DefaultWriteIdempotencyTTL(),
		func(ctx context.Context) (any, error) {
			order, err := h.referralService.CreateRechargeOrder(ctx, &service.CreateRechargeOrderInput{
				UserID:   subject.UserID,
				Amount:   req.Amount,
				Channel:  channel,
				Currency: req.Currency,
				Notes:    req.Notes,
			})
			if err != nil {
				return nil, err
			}

			resp := CreateRechargeOrderResponse{Order: order}
			if channel == service.XunhuPayChannel {
				if h.xunhuPayService == nil {
					return nil, infraerrors.New(http.StatusServiceUnavailable, "XUNHUPAY_SERVICE_UNAVAILABLE", "xunhupay service is unavailable")
				}

				payment, err := h.xunhuPayService.CreatePayment(ctx, &service.XunhuCreatePaymentInput{
					OrderNo: order.OrderNo,
					Amount:  order.Amount,
					Title:   req.Title,
					Attach:  req.Notes,
				})
				if err != nil {
					return nil, markOrderFailedOnPaymentInitialization(ctx, h.referralService, order.OrderNo, channel, err)
				}
				resp.Payment = payment

				if payment.OpenOrderID != "" {
					updatedOrder, updateErr := h.referralService.SetRechargeOrderExternalOrderID(ctx, order.OrderNo, channel, payment.OpenOrderID)
					if updateErr == nil {
						resp.Order = updatedOrder
					}
				}
			}

			return resp, nil
		},
	)
}

func (h *RechargeHandler) ListOrders(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, total, err := h.referralService.ListRechargeOrders(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Paginated(c, items, total, page, pageSize)
}

func (h *RechargeHandler) GetOrder(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderNo := strings.TrimSpace(c.Param("orderNo"))
	order, err := h.referralService.GetRechargeOrder(c.Request.Context(), subject.UserID, orderNo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, order)
}

func (h *RechargeHandler) ReconcileOrder(c *gin.Context) {
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	orderNo := strings.TrimSpace(c.Param("orderNo"))
	if orderNo == "" {
		response.BadRequest(c, "Invalid order number")
		return
	}

	order, err := h.referralService.GetRechargeOrder(c.Request.Context(), subject.UserID, orderNo)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if !strings.EqualFold(strings.TrimSpace(order.Channel), service.XunhuPayChannel) {
		response.BadRequest(c, "Only xunhupay orders support reconcile")
		return
	}
	if h.xunhuPayService == nil {
		response.ErrorFrom(c, infraerrors.New(http.StatusServiceUnavailable, "XUNHUPAY_SERVICE_UNAVAILABLE", "xunhupay service is unavailable"))
		return
	}

	payment, err := h.xunhuPayService.QueryPayment(c.Request.Context(), &service.XunhuQueryPaymentInput{
		OrderNo:     order.OrderNo,
		OpenOrderID: derefStringPtr(order.ExternalOrderID),
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	if payment.OpenOrderID != "" && derefStringPtr(order.ExternalOrderID) == "" {
		if updatedOrder, setErr := h.referralService.SetRechargeOrderExternalOrderID(c.Request.Context(), order.OrderNo, service.XunhuPayChannel, payment.OpenOrderID); setErr == nil {
			order = updatedOrder
		}
	}

	nextStatus := mapXunhuQueryStatusToRechargeStatus(payment.Status, order.Status)
	if nextStatus == "" || strings.EqualFold(nextStatus, order.Status) {
		response.Success(c, ReconcileRechargeOrderResponse{
			Order:   order,
			Payment: payment,
		})
		return
	}

	amount := payment.TotalFee
	currency := strings.TrimSpace(order.Currency)
	if currency == "" {
		currency = "CNY"
	}

	callbackRaw := marshalRechargeRaw(payment.ResponseRaw)
	updatedOrder, commissions, err := h.referralService.HandlePaymentWebhook(c.Request.Context(), &service.PaymentWebhookInput{
		OrderNo:                firstNonEmptyString(payment.OrderNo, order.OrderNo),
		ExternalOrderID:        firstNonEmptyString(payment.OpenOrderID, derefStringPtr(order.ExternalOrderID)),
		Channel:                service.XunhuPayChannel,
		Status:                 nextStatus,
		Amount:                 amount,
		CreditedAmount:         0,
		Currency:               currency,
		CallbackIdempotencyKey: buildXunhuReconcileIdempotencyKey(order, payment, nextStatus),
		CallbackRaw:            callbackRaw,
		Notes:                  "xunhupay reconcile",
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, ReconcileRechargeOrderResponse{
		Order:       updatedOrder,
		Payment:     payment,
		Commissions: commissions,
	})
}

func (h *RechargeHandler) HandleWebhook(c *gin.Context) {
	channel := normalizeRechargeChannel(c.Param("channel"))
	if channel == service.XunhuPayChannel {
		h.handleXunhuPayWebhook(c)
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	if !h.authorizeWebhook(c, rawBody) {
		return
	}

	var req PaymentWebhookRequest
	if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &req); err != nil {
			response.BadRequest(c, "Invalid request: "+err.Error())
			return
		}
	}

	order, commissions, err := h.referralService.HandlePaymentWebhook(c.Request.Context(), &service.PaymentWebhookInput{
		OrderNo:                req.OrderNo,
		ExternalOrderID:        req.ExternalOrderID,
		Channel:                channel,
		Status:                 req.Status,
		Amount:                 req.Amount,
		CreditedAmount:         0,
		Currency:               req.Currency,
		CallbackIdempotencyKey: req.CallbackIdempotencyKey,
		CallbackRaw:            strings.TrimSpace(string(rawBody)),
		Notes:                  req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, gin.H{
		"order":       order,
		"commissions": commissions,
	})
}

func (h *RechargeHandler) handleXunhuPayWebhook(c *gin.Context) {
	if h.xunhuPayService == nil {
		c.String(http.StatusServiceUnavailable, "xunhupay service is unavailable")
		return
	}
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "invalid xunhupay webhook")
		return
	}

	notification, err := h.xunhuPayService.VerifyWebhook(c.Request.Context(), c.Request.PostForm)
	if err != nil {
		writePlainWebhookError(c, err)
		return
	}

	var webhookStatus string
	switch notification.Status {
	case "OD":
		webhookStatus = service.RechargeOrderStatusPaid
	case "CD":
		webhookStatus = service.RechargeOrderStatusRefunded
	case "WP", "RD", "UD", "":
		c.String(http.StatusOK, "success")
		return
	default:
		c.String(http.StatusOK, "success")
		return
	}

	_, _, err = h.referralService.HandlePaymentWebhook(c.Request.Context(), &service.PaymentWebhookInput{
		OrderNo:                notification.TradeOrderID,
		ExternalOrderID:        notification.OpenOrderID,
		Channel:                service.XunhuPayChannel,
		Status:                 webhookStatus,
		Amount:                 notification.TotalFee,
		CreditedAmount:         0,
		Currency:               "CNY",
		CallbackIdempotencyKey: notification.IdempotencyKey,
		CallbackRaw:            notification.Raw,
		Notes:                  firstNonEmptyString(notification.Attach, notification.OrderTitle),
	})
	if err != nil {
		writePlainWebhookError(c, err)
		return
	}

	c.String(http.StatusOK, "success")
}

func normalizeRechargeChannel(channel string) string {
	channel = strings.ToLower(strings.TrimSpace(channel))
	switch channel {
	case "", "custom":
		return channel
	case "xunhu":
		return service.XunhuPayChannel
	default:
		return channel
	}
}

func mapXunhuQueryStatusToRechargeStatus(providerStatus, currentStatus string) string {
	providerStatus = strings.ToUpper(strings.TrimSpace(providerStatus))
	currentStatus = strings.ToLower(strings.TrimSpace(currentStatus))

	switch providerStatus {
	case "OD":
		return service.RechargeOrderStatusPaid
	case "CD":
		if currentStatus == service.RechargeOrderStatusPaid || currentStatus == service.RechargeOrderStatusRefunded {
			return service.RechargeOrderStatusRefunded
		}
		return service.RechargeOrderStatusFailed
	case "WP", "RD", "UD", "":
		return ""
	default:
		return ""
	}
}

func buildXunhuReconcileIdempotencyKey(order *service.RechargeOrder, payment *service.XunhuQueryPaymentResult, status string) string {
	base := ""
	if payment != nil {
		base = firstNonEmptyString(payment.TransactionID, payment.OpenOrderID, payment.OrderNo)
	}
	if base == "" && order != nil {
		base = strings.TrimSpace(order.OrderNo)
	}
	return strings.ToLower(strings.TrimSpace(status)) + ":" + base
}

func marshalRechargeRaw(raw map[string]any) string {
	if len(raw) == 0 {
		return ""
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return ""
	}
	return string(body)
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func writePlainWebhookError(c *gin.Context, err error) {
	statusCode, status := infraerrors.ToHTTP(err)
	c.String(statusCode, status.Message)
}

func (h *RechargeHandler) authorizeWebhook(c *gin.Context, rawBody []byte) bool {
	if h.settingService == nil {
		response.Forbidden(c, "Payment webhook is not configured")
		return false
	}

	timestampRaw := strings.TrimSpace(c.GetHeader(paymentWebhookTimestampHeader))
	if timestampRaw == "" {
		response.Unauthorized(c, "Webhook timestamp required")
		return false
	}
	timestampUnix, err := strconv.ParseInt(timestampRaw, 10, 64)
	if err != nil || timestampUnix <= 0 {
		response.BadRequest(c, "Invalid webhook timestamp")
		return false
	}
	timestamp := time.Unix(timestampUnix, 0)
	now := time.Now()
	if timestamp.Before(now.Add(-paymentWebhookMaxSkew)) || timestamp.After(now.Add(paymentWebhookMaxSkew)) {
		response.Unauthorized(c, "Webhook timestamp expired")
		return false
	}

	incomingSignature := canonicalizeWebhookSignature(c.GetHeader(paymentWebhookSignatureHeader))
	if incomingSignature == "" {
		response.Unauthorized(c, "Webhook signature required")
		return false
	}

	storedKey, err := h.settingService.GetPaymentWebhookAPIKey(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to validate webhook authentication")
		return false
	}
	storedKey = strings.TrimSpace(storedKey)
	if storedKey == "" {
		response.Forbidden(c, "Payment webhook key is not configured")
		return false
	}
	expectedSignature := computePaymentWebhookSignature(timestampRaw, rawBody, storedKey)
	if subtle.ConstantTimeCompare([]byte(incomingSignature), []byte(expectedSignature)) != 1 {
		response.Unauthorized(c, "Invalid webhook signature")
		return false
	}

	return true
}

func canonicalizeWebhookSignature(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "sha256=") {
		value = strings.TrimSpace(value[len("sha256="):])
	}
	value = strings.ToLower(value)
	if _, err := hex.DecodeString(value); err != nil {
		return ""
	}
	return value
}

func computePaymentWebhookSignature(timestamp string, rawBody []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	mac.Write([]byte(strings.TrimSpace(timestamp)))
	mac.Write([]byte{'.'})
	mac.Write(rawBody)
	return hex.EncodeToString(mac.Sum(nil))
}
