package handler

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type ReferralHandler struct {
	referralService *service.ReferralService
}

type referralSummaryResponse struct {
	ReferralCode             string     `json:"referral_code"`
	InviteeCount             int64      `json:"invitee_count"`
	FirstCommissionCount     int64      `json:"first_commission_count"`
	TotalRechargeAmount      float64    `json:"total_recharge_amount"`
	TotalFirstCommission     float64    `json:"total_first_commission"`
	TotalRecurringCommission float64    `json:"total_recurring_commission"`
	TotalCommission          float64    `json:"total_commission"`
	EffectiveInviteeCount    int64      `json:"effective_invitee_count"`
	WithdrawEnabled          bool       `json:"withdraw_enabled"`
	WithdrawMinAmount        float64    `json:"withdraw_min_amount"`
	WithdrawMinInvitees      int64      `json:"withdraw_min_invitees"`
	AvailableCommission      float64    `json:"available_commission"`
	FrozenCommission         float64    `json:"frozen_commission"`
	NextUnlockAt             *time.Time `json:"next_unlock_at,omitempty"`
	PendingWithdrawalAmount  float64    `json:"pending_withdrawal_amount"`
	ApprovedWithdrawalAmount float64    `json:"approved_withdrawal_amount"`
	CanWithdraw              bool       `json:"can_withdraw"`
}

type referralInviteeResponse struct {
	UserID                    int64      `json:"user_id"`
	Email                     string     `json:"email"`
	Username                  string     `json:"username"`
	RegisteredAt              time.Time  `json:"registered_at"`
	FirstPaidAt               *time.Time `json:"first_paid_at,omitempty"`
	FirstPaidAmount           float64    `json:"first_paid_amount"`
	TotalPaidAmount           float64    `json:"total_paid_amount"`
	FirstCommissionAmount     float64    `json:"first_commission_amount"`
	RecurringCommissionAmount float64    `json:"recurring_commission_amount"`
	TotalCommissionAmount     float64    `json:"total_commission_amount"`
}

type referralCommissionResponse struct {
	ID               int64      `json:"id"`
	CommissionType   string     `json:"commission_type"`
	Status           string     `json:"status"`
	SourceAmount     float64    `json:"source_amount"`
	RateSnapshot     float64    `json:"rate_snapshot"`
	CommissionAmount float64    `json:"commission_amount"`
	CreatedAt        time.Time  `json:"created_at"`
	ReferredUserID   int64      `json:"referred_user_id"`
	ReferredEmail    string     `json:"referred_email,omitempty"`
	ReferredUsername string     `json:"referred_username,omitempty"`
	RechargeOrderID  int64      `json:"recharge_order_id"`
	OrderNo          string     `json:"order_no,omitempty"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

type createReferralWithdrawalRequest struct {
	Amount            float64 `json:"amount" binding:"required,gt=0"`
	Currency          string  `json:"currency"`
	PaymentMethod     string  `json:"payment_method" binding:"required"`
	AccountName       string  `json:"account_name"`
	AccountIdentifier string  `json:"account_identifier" binding:"required"`
	Notes             string  `json:"notes"`
}

type referralWithdrawalRequestResponse struct {
	ID                int64      `json:"id"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	PaymentMethod     string     `json:"payment_method"`
	AccountName       string     `json:"account_name,omitempty"`
	AccountIdentifier string     `json:"account_identifier,omitempty"`
	Status            string     `json:"status"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	Notes             string     `json:"notes,omitempty"`
	ReviewNotes       string     `json:"review_notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func NewReferralHandler(referralService *service.ReferralService) *ReferralHandler {
	return &ReferralHandler{referralService: referralService}
}

func (h *ReferralHandler) GetInviteCode(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	code, err := h.referralService.GetOrCreateReferralCode(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"referral_code": code})
}

func (h *ReferralHandler) GetSummary(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	summary, err := h.referralService.GetSummary(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, referralSummaryResponse{
		ReferralCode:             summary.ReferralCode,
		InviteeCount:             summary.InviteeCount,
		FirstCommissionCount:     summary.FirstCommissionCount,
		TotalRechargeAmount:      summary.TotalRechargeAmount,
		TotalFirstCommission:     summary.TotalFirstCommission,
		TotalRecurringCommission: summary.TotalRecurringCommission,
		TotalCommission:          summary.TotalCommission,
		EffectiveInviteeCount:    summary.EffectiveInviteeCount,
		WithdrawEnabled:          summary.WithdrawEnabled,
		WithdrawMinAmount:        summary.WithdrawMinAmount,
		WithdrawMinInvitees:      summary.WithdrawMinInvitees,
		AvailableCommission:      summary.AvailableCommission,
		FrozenCommission:         summary.FrozenCommission,
		NextUnlockAt:             summary.NextUnlockAt,
		PendingWithdrawalAmount:  summary.PendingWithdrawalAmount,
		ApprovedWithdrawalAmount: summary.ApprovedWithdrawalAmount,
		CanWithdraw:              summary.CanWithdraw,
	})
}

func (h *ReferralHandler) ListInvitees(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, total, err := h.referralService.ListInvitees(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]referralInviteeResponse, 0, len(items))
	for _, item := range items {
		out = append(out, referralInviteeResponse{
			UserID:                    item.UserID,
			Email:                     item.Email,
			Username:                  item.Username,
			RegisteredAt:              item.RegisteredAt,
			FirstPaidAt:               item.FirstPaidAt,
			FirstPaidAmount:           item.FirstPaidAmount,
			TotalPaidAmount:           item.TotalPaidAmount,
			FirstCommissionAmount:     item.FirstCommissionAmount,
			RecurringCommissionAmount: item.RecurringCommissionAmount,
			TotalCommissionAmount:     item.TotalCommissionAmount,
		})
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *ReferralHandler) CreateWithdrawalRequest(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req createReferralWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	item, err := h.referralService.CreateWithdrawalRequest(c.Request.Context(), &service.CreateReferralWithdrawalInput{
		UserID:            subject.UserID,
		Amount:            req.Amount,
		Currency:          req.Currency,
		PaymentMethod:     req.PaymentMethod,
		AccountName:       req.AccountName,
		AccountIdentifier: req.AccountIdentifier,
		Notes:             req.Notes,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, toReferralWithdrawalRequestResponse(item))
}

func (h *ReferralHandler) ListWithdrawalRequests(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, total, err := h.referralService.ListWithdrawalRequests(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]referralWithdrawalRequestResponse, 0, len(items))
	for idx := range items {
		item := items[idx]
		out = append(out, toReferralWithdrawalRequestResponse(&item))
	}
	response.Paginated(c, out, total, page, pageSize)
}

func (h *ReferralHandler) CancelWithdrawalRequest(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	requestID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || requestID <= 0 {
		response.BadRequest(c, "Invalid withdrawal request id")
		return
	}

	item, err := h.referralService.CancelWithdrawalRequest(c.Request.Context(), subject.UserID, requestID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, toReferralWithdrawalRequestResponse(item))
}

func toReferralWithdrawalRequestResponse(item *service.ReferralWithdrawalRequest) referralWithdrawalRequestResponse {
	if item == nil {
		return referralWithdrawalRequestResponse{}
	}
	return referralWithdrawalRequestResponse{
		ID:                item.ID,
		Amount:            item.Amount,
		Currency:          item.Currency,
		PaymentMethod:     item.PaymentMethod,
		AccountName:       derefString(item.AccountName),
		AccountIdentifier: derefString(item.AccountIdentifier),
		Status:            item.Status,
		ReviewedAt:        item.ReviewedAt,
		Notes:             derefString(item.Notes),
		ReviewNotes:       derefString(item.ReviewNotes),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (h *ReferralHandler) ListCommissions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	page, pageSize := response.ParsePagination(c)
	items, total, err := h.referralService.ListCommissions(c.Request.Context(), subject.UserID, page, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]referralCommissionResponse, 0, len(items))
	for _, item := range items {
		entry := referralCommissionResponse{
			ID:               item.Commission.ID,
			CommissionType:   item.Commission.CommissionType,
			Status:           item.Commission.Status,
			SourceAmount:     item.Commission.SourceAmount,
			RateSnapshot:     item.Commission.RateSnapshot,
			CommissionAmount: item.Commission.CommissionAmount,
			CreatedAt:        item.Commission.CreatedAt,
			ReferredUserID:   item.Commission.ReferredUserID,
			RechargeOrderID:  item.Commission.RechargeOrderID,
		}
		if item.ReferredUser != nil {
			entry.ReferredEmail = item.ReferredUser.Email
			entry.ReferredUsername = item.ReferredUser.Username
		}
		if item.RechargeOrder != nil {
			entry.OrderNo = item.RechargeOrder.OrderNo
			entry.PaidAt = item.RechargeOrder.PaidAt
		}
		out = append(out, entry)
	}
	response.Paginated(c, out, total, page, pageSize)
}
