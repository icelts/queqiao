package admin

import (
	"strconv"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	referralService *service.ReferralService
}

type RecordPaidRechargeRequest struct {
	UserID                 int64   `json:"user_id" binding:"required"`
	OrderNo                string  `json:"order_no" binding:"required"`
	ExternalOrderID        string  `json:"external_order_id"`
	Channel                string  `json:"channel"`
	Amount                 float64 `json:"amount" binding:"required,gt=0"`
	CreditedAmount         float64 `json:"credited_amount"`
	Currency               string  `json:"currency"`
	CallbackIdempotencyKey string  `json:"callback_idempotency_key"`
	CallbackRaw            string  `json:"callback_raw"`
	Notes                  string  `json:"notes"`
}

type referralCommissionRecord struct {
	ID               int64     `json:"id"`
	PromoterUserID   int64     `json:"promoter_user_id"`
	ReferredUserID   int64     `json:"referred_user_id"`
	RechargeOrderID  int64     `json:"recharge_order_id"`
	CommissionType   string    `json:"commission_type"`
	Status           string    `json:"status"`
	SourceAmount     float64   `json:"source_amount"`
	RateSnapshot     float64   `json:"rate_snapshot"`
	CommissionAmount float64   `json:"commission_amount"`
	Currency         string    `json:"currency"`
	CreatedAt        time.Time `json:"created_at"`
}

type referralWithdrawalRecord struct {
	ID                int64      `json:"id"`
	PromoterUserID    int64      `json:"promoter_user_id"`
	PromoterEmail     string     `json:"promoter_email,omitempty"`
	PromoterUsername  string     `json:"promoter_username,omitempty"`
	ReviewerUserID    *int64     `json:"reviewer_user_id,omitempty"`
	ReviewerEmail     string     `json:"reviewer_email,omitempty"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	PaymentMethod     string     `json:"payment_method"`
	AccountName       string     `json:"account_name,omitempty"`
	AccountIdentifier string     `json:"account_identifier,omitempty"`
	Status            string     `json:"status"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	ReviewNotes       string     `json:"review_notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type reviewReferralWithdrawalRequest struct {
	ReviewNotes string `json:"review_notes"`
}

type adminRechargeOrderRecord struct {
	ID                       int64      `json:"id"`
	UserID                   int64      `json:"user_id"`
	UserEmail                string     `json:"user_email,omitempty"`
	Username                 string     `json:"username,omitempty"`
	OrderNo                  string     `json:"order_no"`
	ExternalOrderID          string     `json:"external_order_id,omitempty"`
	Channel                  string     `json:"channel"`
	Currency                 string     `json:"currency"`
	Amount                   float64    `json:"amount"`
	CreditedAmount           float64    `json:"credited_amount"`
	Status                   string     `json:"status"`
	PaidAt                   *time.Time `json:"paid_at,omitempty"`
	RefundedAt               *time.Time `json:"refunded_at,omitempty"`
	CallbackIdempotencyKey   string     `json:"callback_idempotency_key,omitempty"`
	Notes                    string     `json:"notes,omitempty"`
	CommissionCount          int        `json:"commission_count"`
	TotalCommissionAmount    float64    `json:"total_commission_amount"`
	RecordedCommissionAmount float64    `json:"recorded_commission_amount"`
	ReversedCommissionAmount float64    `json:"reversed_commission_amount"`
	CreatedAt                time.Time  `json:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at"`
}

type adminRechargeOrderStatsResponse struct {
	TotalOrders           int64   `json:"total_orders"`
	PendingOrders         int64   `json:"pending_orders"`
	PaidOrders            int64   `json:"paid_orders"`
	FailedOrders          int64   `json:"failed_orders"`
	RefundedOrders        int64   `json:"refunded_orders"`
	TotalPaidAmount       float64 `json:"total_paid_amount"`
	TotalRefundedAmount   float64 `json:"total_refunded_amount"`
	TotalCommissionAmount float64 `json:"total_commission_amount"`
}

type adminRechargeOrderDetailResponse struct {
	Order       adminRechargeOrderRecord `json:"order"`
	CallbackRaw string                   `json:"callback_raw,omitempty"`
}

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

func (h *ReferralHandler) RecordPaidRecharge(c *gin.Context) {
	var req RecordPaidRechargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	order, commissions, err := h.referralService.RecordPaidRecharge(c.Request.Context(), &service.RecordPaidRechargeInput{
		UserID:                 req.UserID,
		OrderNo:                req.OrderNo,
		ExternalOrderID:        req.ExternalOrderID,
		Channel:                req.Channel,
		Amount:                 req.Amount,
		CreditedAmount:         req.CreditedAmount,
		Currency:               req.Currency,
		CallbackIdempotencyKey: req.CallbackIdempotencyKey,
		CallbackRaw:            req.CallbackRaw,
		Notes:                  req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	outCommissions := make([]referralCommissionRecord, 0, len(commissions))
	for _, item := range commissions {
		outCommissions = append(outCommissions, referralCommissionRecord{
			ID:               item.ID,
			PromoterUserID:   item.PromoterUserID,
			ReferredUserID:   item.ReferredUserID,
			RechargeOrderID:  item.RechargeOrderID,
			CommissionType:   item.CommissionType,
			Status:           item.Status,
			SourceAmount:     item.SourceAmount,
			RateSnapshot:     item.RateSnapshot,
			CommissionAmount: item.CommissionAmount,
			Currency:         item.Currency,
			CreatedAt:        item.CreatedAt,
		})
	}

	response.Success(c, gin.H{
		"order":       order,
		"commissions": outCommissions,
	})
}

func (h *ReferralHandler) ListRechargeOrders(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	startDate, endDate, err := parseAdminDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	channel := strings.TrimSpace(c.Query("channel"))
	search := strings.TrimSpace(c.Query("search"))
	withCommission := c.Query("with_commission") == "true"
	refundedOnly := c.Query("refunded_only") == "true"
	if len(search) > 100 {
		search = search[:100]
	}

	items, total, err := h.referralService.ListAdminRechargeOrders(c.Request.Context(), &service.AdminListRechargeOrdersInput{
		Page:           page,
		PageSize:       pageSize,
		Status:         status,
		Channel:        channel,
		Search:         search,
		StartDate:      startDate,
		EndDate:        endDate,
		WithCommission: withCommission,
		RefundedOnly:   refundedOnly,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]adminRechargeOrderRecord, 0, len(items))
	for _, item := range items {
		record := adminRechargeOrderRecord{
			ID:                       item.RechargeOrder.ID,
			UserID:                   item.RechargeOrder.UserID,
			OrderNo:                  item.RechargeOrder.OrderNo,
			ExternalOrderID:          derefString(item.RechargeOrder.ExternalOrderID),
			Channel:                  item.RechargeOrder.Channel,
			Currency:                 item.RechargeOrder.Currency,
			Amount:                   item.RechargeOrder.Amount,
			CreditedAmount:           item.RechargeOrder.CreditedAmount,
			Status:                   item.RechargeOrder.Status,
			PaidAt:                   item.RechargeOrder.PaidAt,
			RefundedAt:               item.RechargeOrder.RefundedAt,
			CallbackIdempotencyKey:   item.RechargeOrder.CallbackIdempotencyKey,
			Notes:                    derefString(item.RechargeOrder.Notes),
			CommissionCount:          item.CommissionCount,
			TotalCommissionAmount:    item.TotalCommissionAmount,
			RecordedCommissionAmount: item.RecordedCommissionAmount,
			ReversedCommissionAmount: item.ReversedCommissionAmount,
			CreatedAt:                item.RechargeOrder.CreatedAt,
			UpdatedAt:                item.RechargeOrder.UpdatedAt,
		}
		if item.User != nil {
			record.UserEmail = item.User.Email
			record.Username = item.User.Username
		}
		out = append(out, record)
	}

	response.Paginated(c, out, total, page, pageSize)
}

func (h *ReferralHandler) GetRechargeOrderStats(c *gin.Context) {
	startDate, endDate, err := parseAdminDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	stats, err := h.referralService.GetAdminRechargeOrderStats(c.Request.Context(), &service.AdminListRechargeOrdersInput{
		Status:         strings.TrimSpace(c.Query("status")),
		Channel:        strings.TrimSpace(c.Query("channel")),
		Search:         strings.TrimSpace(c.Query("search")),
		StartDate:      startDate,
		EndDate:        endDate,
		WithCommission: c.Query("with_commission") == "true",
		RefundedOnly:   c.Query("refunded_only") == "true",
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, adminRechargeOrderStatsResponse{
		TotalOrders:           stats.TotalOrders,
		PendingOrders:         stats.PendingOrders,
		PaidOrders:            stats.PaidOrders,
		FailedOrders:          stats.FailedOrders,
		RefundedOrders:        stats.RefundedOrders,
		TotalPaidAmount:       stats.TotalPaidAmount,
		TotalRefundedAmount:   stats.TotalRefundedAmount,
		TotalCommissionAmount: stats.TotalCommissionAmount,
	})
}

func (h *ReferralHandler) GetRechargeOrderDetail(c *gin.Context) {
	orderID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || orderID <= 0 {
		response.BadRequest(c, "Invalid recharge order id")
		return
	}

	item, err := h.referralService.GetAdminRechargeOrderDetail(c.Request.Context(), orderID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	record := adminRechargeOrderRecord{
		ID:                       item.RechargeOrder.ID,
		UserID:                   item.RechargeOrder.UserID,
		UserEmail:                "",
		Username:                 "",
		OrderNo:                  item.RechargeOrder.OrderNo,
		ExternalOrderID:          derefString(item.RechargeOrder.ExternalOrderID),
		Channel:                  item.RechargeOrder.Channel,
		Currency:                 item.RechargeOrder.Currency,
		Amount:                   item.RechargeOrder.Amount,
		CreditedAmount:           item.RechargeOrder.CreditedAmount,
		Status:                   item.RechargeOrder.Status,
		PaidAt:                   item.RechargeOrder.PaidAt,
		RefundedAt:               item.RechargeOrder.RefundedAt,
		CallbackIdempotencyKey:   item.RechargeOrder.CallbackIdempotencyKey,
		Notes:                    derefString(item.RechargeOrder.Notes),
		CommissionCount:          item.CommissionCount,
		TotalCommissionAmount:    item.TotalCommissionAmount,
		RecordedCommissionAmount: item.RecordedCommissionAmount,
		ReversedCommissionAmount: item.ReversedCommissionAmount,
		CreatedAt:                item.RechargeOrder.CreatedAt,
		UpdatedAt:                item.RechargeOrder.UpdatedAt,
	}
	if item.User != nil {
		record.UserEmail = item.User.Email
		record.Username = item.User.Username
	}

	response.Success(c, adminRechargeOrderDetailResponse{
		Order:       record,
		CallbackRaw: derefString(item.RechargeOrder.CallbackRaw),
	})
}

func (h *ReferralHandler) ListWithdrawalRequests(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := c.Query("status")

	items, total, err := h.referralService.ListAdminWithdrawalRequests(c.Request.Context(), status, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]referralWithdrawalRecord, 0, len(items))
	for _, item := range items {
		record := referralWithdrawalRecord{
			ID:                item.Request.ID,
			PromoterUserID:    item.Request.PromoterUserID,
			ReviewerUserID:    item.Request.ReviewerUserID,
			Amount:            item.Request.Amount,
			Currency:          item.Request.Currency,
			PaymentMethod:     item.Request.PaymentMethod,
			AccountName:       derefString(item.Request.AccountName),
			AccountIdentifier: derefString(item.Request.AccountIdentifier),
			Status:            item.Request.Status,
			ReviewedAt:        item.Request.ReviewedAt,
			PaidAt:            item.Request.PaidAt,
			Notes:             derefString(item.Request.Notes),
			ReviewNotes:       derefString(item.Request.ReviewNotes),
			CreatedAt:         item.Request.CreatedAt,
			UpdatedAt:         item.Request.UpdatedAt,
		}
		if item.Promoter != nil {
			record.PromoterEmail = item.Promoter.Email
			record.PromoterUsername = item.Promoter.Username
		}
		if item.Reviewer != nil {
			record.ReviewerEmail = item.Reviewer.Email
		}
		out = append(out, record)
	}

	response.Paginated(c, out, total, page, pageSize)
}

func (h *ReferralHandler) ApproveWithdrawalRequest(c *gin.Context) {
	h.reviewWithdrawalRequest(c, true)
}

func (h *ReferralHandler) RejectWithdrawalRequest(c *gin.Context) {
	h.reviewWithdrawalRequest(c, false)
}

func (h *ReferralHandler) MarkWithdrawalRequestPaid(c *gin.Context) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || requestID <= 0 {
		response.BadRequest(c, "Invalid withdrawal request id")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req reviewReferralWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.referralService.MarkWithdrawalRequestPaid(c.Request.Context(), &service.MarkReferralWithdrawalPaidInput{
		RequestID:      requestID,
		OperatorUserID: subject.UserID,
		PaymentNotes:   req.ReviewNotes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, referralWithdrawalRecord{
		ID:                item.ID,
		PromoterUserID:    item.PromoterUserID,
		ReviewerUserID:    item.ReviewerUserID,
		Amount:            item.Amount,
		Currency:          item.Currency,
		PaymentMethod:     item.PaymentMethod,
		AccountName:       derefString(item.AccountName),
		AccountIdentifier: derefString(item.AccountIdentifier),
		Status:            item.Status,
		ReviewedAt:        item.ReviewedAt,
		PaidAt:            item.PaidAt,
		Notes:             derefString(item.Notes),
		ReviewNotes:       derefString(item.ReviewNotes),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

func (h *ReferralHandler) reviewWithdrawalRequest(c *gin.Context, approve bool) {
	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || requestID <= 0 {
		response.BadRequest(c, "Invalid withdrawal request id")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req reviewReferralWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	input := &service.ReviewReferralWithdrawalInput{
		RequestID:      requestID,
		ReviewerUserID: subject.UserID,
		ReviewNotes:    req.ReviewNotes,
	}

	var item *service.ReferralWithdrawalRequest
	if approve {
		item, err = h.referralService.ApproveWithdrawalRequest(c.Request.Context(), input)
	} else {
		item, err = h.referralService.RejectWithdrawalRequest(c.Request.Context(), input)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, referralWithdrawalRecord{
		ID:                item.ID,
		PromoterUserID:    item.PromoterUserID,
		ReviewerUserID:    item.ReviewerUserID,
		Amount:            item.Amount,
		Currency:          item.Currency,
		PaymentMethod:     item.PaymentMethod,
		AccountName:       derefString(item.AccountName),
		AccountIdentifier: derefString(item.AccountIdentifier),
		Status:            item.Status,
		ReviewedAt:        item.ReviewedAt,
		PaidAt:            item.PaidAt,
		Notes:             derefString(item.Notes),
		ReviewNotes:       derefString(item.ReviewNotes),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	})
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func parseAdminDateRange(start, end string) (*time.Time, *time.Time, error) {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" && end == "" {
		return nil, nil, nil
	}

	parse := func(value string) (*time.Time, error) {
		if value == "" {
			return nil, nil
		}
		t, err := time.Parse("2006-01-02", value)
		if err != nil {
			return nil, err
		}
		return &t, nil
	}

	startAt, err := parse(start)
	if err != nil {
		return nil, nil, infraerrors.BadRequest("INVALID_DATE_RANGE", "invalid start_date")
	}
	endAt, err := parse(end)
	if err != nil {
		return nil, nil, infraerrors.BadRequest("INVALID_DATE_RANGE", "invalid end_date")
	}
	return startAt, endAt, nil
}
