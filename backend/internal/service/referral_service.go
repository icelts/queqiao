package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrRechargeOrderInvalid      = infraerrors.BadRequest("RECHARGE_ORDER_INVALID", "invalid recharge order")
	ErrRechargeOrderUserMismatch = infraerrors.Conflict("RECHARGE_ORDER_USER_MISMATCH", "recharge order belongs to another user")
)

type RechargeOrder struct {
	ID                     int64      `json:"id"`
	UserID                 int64      `json:"user_id"`
	OrderNo                string     `json:"order_no"`
	ExternalOrderID        *string    `json:"external_order_id,omitempty"`
	Channel                string     `json:"channel"`
	Source                 string     `json:"source"`
	Currency               string     `json:"currency"`
	Amount                 float64    `json:"amount"`
	CreditedAmount         float64    `json:"credited_amount"`
	Status                 string     `json:"status"`
	PaidAt                 *time.Time `json:"paid_at,omitempty"`
	RefundedAt             *time.Time `json:"refunded_at,omitempty"`
	CallbackIdempotencyKey string     `json:"callback_idempotency_key,omitempty"`
	CallbackRaw            *string    `json:"callback_raw,omitempty"`
	Notes                  *string    `json:"notes,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type ReferralCommission struct {
	ID               int64      `json:"id"`
	PromoterUserID   int64      `json:"promoter_user_id"`
	ReferredUserID   int64      `json:"referred_user_id"`
	RechargeOrderID  int64      `json:"recharge_order_id"`
	CommissionType   string     `json:"commission_type"`
	Status           string     `json:"status"`
	SourceAmount     float64    `json:"source_amount"`
	RateSnapshot     float64    `json:"rate_snapshot"`
	CommissionAmount float64    `json:"commission_amount"`
	Currency         string     `json:"currency"`
	ReversedAt       *time.Time `json:"reversed_at,omitempty"`
	ReversedReason   *string    `json:"reversed_reason,omitempty"`
	Notes            *string    `json:"notes,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ReferralSummary struct {
	ReferralCode                 string
	InviteeCount                 int64
	FirstCommissionCount         int64
	TotalRechargeAmount          float64
	TotalFirstCommission         float64
	TotalRecurringCommission     float64
	TotalCommission              float64
	EffectiveInviteeCount        int64
	WithdrawEnabled              bool
	WithdrawMinAmount            float64
	WithdrawMinInvitees          int64
	CommissionCurrency           string
	HasMixedCommissionCurrencies bool
	AvailableCommission          float64
	FrozenCommission             float64
	NextUnlockAt                 *time.Time
	PendingWithdrawalAmount      float64
	ApprovedWithdrawalAmount     float64
	PaidWithdrawalAmount         float64
	WithdrawalDebt               float64
	CanWithdraw                  bool
}

type ReferralInvitee struct {
	UserID                    int64
	Email                     string
	Username                  string
	RegisteredAt              time.Time
	FirstPaidAt               *time.Time
	FirstPaidAmount           float64
	TotalPaidAmount           float64
	FirstCommissionAmount     float64
	RecurringCommissionAmount float64
	TotalCommissionAmount     float64
}

type ReferralCommissionDetail struct {
	Commission    ReferralCommission
	ReferredUser  *User
	RechargeOrder *RechargeOrder
}

type ReferralWithdrawalRequest struct {
	ID                int64      `json:"id"`
	PromoterUserID    int64      `json:"promoter_user_id"`
	ReviewerUserID    *int64     `json:"reviewer_user_id,omitempty"`
	Amount            float64    `json:"amount"`
	Currency          string     `json:"currency"`
	PaymentMethod     string     `json:"payment_method"`
	AccountName       *string    `json:"account_name,omitempty"`
	AccountIdentifier *string    `json:"account_identifier,omitempty"`
	Status            string     `json:"status"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	PaidAt            *time.Time `json:"paid_at,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	ReviewNotes       *string    `json:"review_notes,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ReferralWithdrawalRequestDetail struct {
	Request  ReferralWithdrawalRequest
	Promoter *User
	Reviewer *User
}

type AdminRechargeOrderDetail struct {
	RechargeOrder            RechargeOrder
	User                     *User
	CommissionCount          int
	TotalCommissionAmount    float64
	RecordedCommissionAmount float64
	ReversedCommissionAmount float64
}

type AdminRechargeOrderStats struct {
	TotalOrders           int64
	PendingOrders         int64
	PaidOrders            int64
	FailedOrders          int64
	RefundedOrders        int64
	TotalPaidAmount       float64
	TotalRefundedAmount   float64
	TotalCommissionAmount float64
}

type RecordPaidRechargeInput struct {
	UserID                 int64
	OrderNo                string
	ExternalOrderID        string
	Channel                string
	Amount                 float64
	CreditedAmount         float64
	Currency               string
	CallbackIdempotencyKey string
	CallbackRaw            string
	Notes                  string
}

type ReferralService struct {
	entClient            *dbent.Client
	settingService       *SettingService
	subscriptionService  *SubscriptionService
	billingCacheService  *BillingCacheService
	authCacheInvalidator APIKeyAuthCacheInvalidator
}

func NewReferralService(
	entClient *dbent.Client,
	settingService *SettingService,
	subscriptionService *SubscriptionService,
	billingCacheService *BillingCacheService,
	authCacheInvalidator APIKeyAuthCacheInvalidator,
) *ReferralService {
	return &ReferralService{
		entClient:            entClient,
		settingService:       settingService,
		subscriptionService:  subscriptionService,
		billingCacheService:  billingCacheService,
		authCacheInvalidator: authCacheInvalidator,
	}
}

func (s *ReferralService) GetOrCreateReferralCode(ctx context.Context, userID int64) (string, error) {
	if s.entClient == nil {
		return "", ErrServiceUnavailable
	}

	u, err := s.entClient.User.Query().
		Where(dbuser.IDEQ(userID), dbuser.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	if strings.TrimSpace(u.ReferralCode) != "" {
		return u.ReferralCode, nil
	}

	for i := 0; i < 8; i++ {
		code, err := randomHexString(4)
		if err != nil {
			return "", err
		}
		code = strings.ToUpper(code)
		if _, err := s.entClient.User.Update().
			Where(dbuser.IDEQ(userID), dbuser.ReferralCodeEQ(""), dbuser.DeletedAtIsNil()).
			SetReferralCode(code).
			Save(ctx); err == nil {
			return code, nil
		}
		u, err = s.entClient.User.Query().
			Where(dbuser.IDEQ(userID), dbuser.DeletedAtIsNil()).
			Only(ctx)
		if err == nil && strings.TrimSpace(u.ReferralCode) != "" {
			return u.ReferralCode, nil
		}
	}

	return "", infraerrors.Conflict("REFERRAL_CODE_GENERATION_FAILED", "failed to allocate referral code")
}

func (s *ReferralService) GetSummary(ctx context.Context, promoterUserID int64) (*ReferralSummary, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}

	referralCode, err := s.GetOrCreateReferralCode(ctx, promoterUserID)
	if err != nil {
		return nil, err
	}

	inviteeCount, err := s.entClient.User.Query().
		Where(dbuser.InviterIDEQ(promoterUserID), dbuser.DeletedAtIsNil()).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	withdrawalStats, err := s.getWithdrawalStats(ctx, promoterUserID)
	if err != nil {
		return nil, err
	}

	firstCommissionCount, totalFirstCommission, totalRecurringCommission, totalCommission, err := s.getCommissionSummaryStats(ctx, promoterUserID, withdrawalStats)
	if err != nil {
		return nil, err
	}
	totalRechargeAmount, err := s.getTotalPaidRechargeAmount(ctx, promoterUserID, withdrawalStats)
	if err != nil {
		return nil, err
	}

	summary := &ReferralSummary{
		ReferralCode:                 referralCode,
		InviteeCount:                 int64(inviteeCount),
		FirstCommissionCount:         firstCommissionCount,
		TotalRechargeAmount:          totalRechargeAmount,
		TotalFirstCommission:         totalFirstCommission,
		TotalRecurringCommission:     totalRecurringCommission,
		TotalCommission:              totalCommission,
		WithdrawEnabled:              withdrawalStats.WithdrawEnabled,
		WithdrawMinAmount:            withdrawalStats.WithdrawMinAmount,
		WithdrawMinInvitees:          withdrawalStats.WithdrawMinInvitees,
		CommissionCurrency:           withdrawalStats.Currency,
		HasMixedCommissionCurrencies: withdrawalStats.HasMixedCurrencies,
		AvailableCommission:          withdrawalStats.AvailableCommission,
		FrozenCommission:             withdrawalStats.FrozenCommission,
		NextUnlockAt:                 withdrawalStats.NextUnlockAt,
		PendingWithdrawalAmount:      withdrawalStats.PendingWithdrawalAmount,
		ApprovedWithdrawalAmount:     withdrawalStats.ApprovedWithdrawalAmount,
		PaidWithdrawalAmount:         withdrawalStats.PaidWithdrawalAmount,
		WithdrawalDebt:               withdrawalStats.WithdrawalDebt,
		EffectiveInviteeCount:        withdrawalStats.EffectiveInviteeCount,
		CanWithdraw:                  withdrawalStats.CanWithdraw,
	}

	return summary, nil
}

func (s *ReferralService) ListInvitees(ctx context.Context, promoterUserID int64, page, pageSize int) ([]ReferralInvitee, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}

	page, pageSize = normalizePage(page, pageSize)
	baseQuery := s.entClient.User.Query().
		Where(dbuser.InviterIDEQ(promoterUserID), dbuser.DeletedAtIsNil())

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := baseQuery.
		Order(dbent.Desc(dbuser.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(items) == 0 {
		return []ReferralInvitee{}, int64(total), nil
	}

	result := make([]ReferralInvitee, 0, len(items))
	inviteeIDs := make([]int64, 0, len(items))
	resultIndexByUserID := make(map[int64]int, len(items))
	for _, invitee := range items {
		out := ReferralInvitee{
			UserID:       invitee.ID,
			Email:        invitee.Email,
			Username:     invitee.Username,
			RegisteredAt: invitee.CreatedAt,
		}
		inviteeIDs = append(inviteeIDs, invitee.ID)
		resultIndexByUserID[invitee.ID] = len(result)
		result = append(result, out)
	}

	orders, err := s.entClient.RechargeOrder.Query().
		Where(
			rechargeorder.UserIDIn(inviteeIDs...),
			rechargeorder.StatusEQ(RechargeOrderStatusPaid),
		).
		Order(
			dbent.Asc(rechargeorder.FieldUserID),
			rechargeorder.ByPaidAt(),
			dbent.Asc(rechargeorder.FieldID),
		).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, order := range orders {
		idx, ok := resultIndexByUserID[order.UserID]
		if !ok {
			continue
		}
		result[idx].TotalPaidAmount = roundMoney(result[idx].TotalPaidAmount + order.Amount)
		if result[idx].FirstPaidAt == nil {
			result[idx].FirstPaidAt = order.PaidAt
			result[idx].FirstPaidAmount = order.Amount
		}
	}

	commissions, err := s.entClient.ReferralCommission.Query().
		Where(
			referralcommission.PromoterUserIDEQ(promoterUserID),
			referralcommission.ReferredUserIDIn(inviteeIDs...),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
		).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	for _, commission := range commissions {
		idx, ok := resultIndexByUserID[commission.ReferredUserID]
		if !ok {
			continue
		}
		result[idx].TotalCommissionAmount = roundMoney(result[idx].TotalCommissionAmount + commission.CommissionAmount)
		if commission.CommissionType == ReferralCommissionTypeFirst {
			result[idx].FirstCommissionAmount = roundMoney(result[idx].FirstCommissionAmount + commission.CommissionAmount)
		} else if commission.CommissionType == ReferralCommissionTypeRecurring {
			result[idx].RecurringCommissionAmount = roundMoney(result[idx].RecurringCommissionAmount + commission.CommissionAmount)
		}
	}

	return result, int64(total), nil
}

func (s *ReferralService) getCommissionSummaryStats(ctx context.Context, promoterUserID int64, withdrawalStats *referralWithdrawalStats) (int64, float64, float64, float64, error) {
	firstCommissionCount, err := s.entClient.ReferralCommission.Query().
		Where(
			referralcommission.PromoterUserIDEQ(promoterUserID),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
			referralcommission.CommissionTypeEQ(ReferralCommissionTypeFirst),
		).
		Count(ctx)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	if withdrawalStats == nil || withdrawalStats.HasMixedCurrencies {
		return int64(firstCommissionCount), 0, 0, 0, nil
	}

	var rows []struct {
		CommissionType string  `json:"commission_type"`
		Total          float64 `json:"total"`
	}
	err = s.entClient.ReferralCommission.Query().
		Where(
			referralcommission.PromoterUserIDEQ(promoterUserID),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
			referralcommission.CurrencyEQ(withdrawalStats.Currency),
		).
		GroupBy(referralcommission.FieldCommissionType).
		Aggregate(dbent.As(dbent.Sum(referralcommission.FieldCommissionAmount), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	totalFirstCommission := 0.0
	totalRecurringCommission := 0.0
	totalCommission := 0.0
	for _, row := range rows {
		totalCommission = roundMoney(totalCommission + row.Total)
		switch row.CommissionType {
		case ReferralCommissionTypeFirst:
			totalFirstCommission = roundMoney(totalFirstCommission + row.Total)
		case ReferralCommissionTypeRecurring:
			totalRecurringCommission = roundMoney(totalRecurringCommission + row.Total)
		}
	}

	return int64(firstCommissionCount), totalFirstCommission, totalRecurringCommission, totalCommission, nil
}

func (s *ReferralService) getTotalPaidRechargeAmount(ctx context.Context, promoterUserID int64, withdrawalStats *referralWithdrawalStats) (float64, error) {
	if withdrawalStats == nil || withdrawalStats.HasMixedCurrencies {
		return 0, nil
	}

	var rows []struct {
		Total float64 `json:"total"`
	}
	err := s.entClient.RechargeOrder.Query().
		Where(
			rechargeorder.StatusEQ(RechargeOrderStatusPaid),
			rechargeorder.CurrencyEQ(withdrawalStats.Currency),
			rechargeorder.HasUserWith(
				dbuser.InviterIDEQ(promoterUserID),
				dbuser.DeletedAtIsNil(),
			),
		).
		Aggregate(dbent.As(dbent.Sum(rechargeorder.FieldAmount), "total")).
		Scan(ctx, &rows)
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	return roundMoney(rows[0].Total), nil
}

func (s *ReferralService) ListCommissions(ctx context.Context, promoterUserID int64, page, pageSize int) ([]ReferralCommissionDetail, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}

	page, pageSize = normalizePage(page, pageSize)
	baseQuery := s.entClient.ReferralCommission.Query().
		Where(referralcommission.PromoterUserIDEQ(promoterUserID))

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := baseQuery.
		Order(dbent.Desc(referralcommission.FieldID)).
		WithReferredUser().
		WithRechargeOrder().
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]ReferralCommissionDetail, 0, len(items))
	for _, item := range items {
		detail := ReferralCommissionDetail{
			Commission:    referralCommissionEntityToService(item),
			ReferredUser:  entUserToService(item.Edges.ReferredUser),
			RechargeOrder: rechargeOrderEntityToService(item.Edges.RechargeOrder),
		}
		result = append(result, detail)
	}

	return result, int64(total), nil
}

func (s *ReferralService) RecordPaidRecharge(ctx context.Context, input *RecordPaidRechargeInput) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if input == nil || input.UserID <= 0 || strings.TrimSpace(input.OrderNo) == "" || input.Amount <= 0 {
		return nil, nil, ErrRechargeOrderInvalid
	}

	input.OrderNo = strings.TrimSpace(input.OrderNo)
	input.ExternalOrderID = strings.TrimSpace(input.ExternalOrderID)
	input.Channel = strings.TrimSpace(input.Channel)
	input.Currency = strings.TrimSpace(input.Currency)
	input.CallbackIdempotencyKey = strings.TrimSpace(input.CallbackIdempotencyKey)
	input.CallbackRaw = strings.TrimSpace(input.CallbackRaw)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Channel == "" {
		input.Channel = "manual"
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.CreditedAmount <= 0 {
		input.CreditedAmount = input.Amount
	}

	existingOrder, existingCommissions, err := s.findExistingPaidRecharge(ctx, input.OrderNo, input.Channel, input.ExternalOrderID)
	if err != nil {
		return nil, nil, err
	}
	if existingOrder != nil {
		if existingOrder.UserID != input.UserID {
			return nil, nil, ErrRechargeOrderUserMismatch
		}
		return existingOrder, existingCommissions, nil
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	referredUserQuery := tx.User.Query().
		Where(dbuser.IDEQ(input.UserID), dbuser.DeletedAtIsNil()).
		ForUpdate()
	referredUser, err := referredUserQuery.Only(ctx)
	if err != nil && isForUpdateUnsupportedError(err) {
		referredUser, err = tx.User.Query().
			Where(dbuser.IDEQ(input.UserID), dbuser.DeletedAtIsNil()).
			Only(ctx)
	}
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil, ErrUserNotFound
		}
		return nil, nil, err
	}
	hadSuccessfulRechargeBefore := referredUser.HasSuccessfulRecharge

	orderEntity, err := s.upsertRechargeOrderTx(ctx, tx, input)
	if err != nil {
		return nil, nil, err
	}
	orderEntity, transitionedToPaid, err := s.markRechargeOrderPaidTx(ctx, tx, orderEntity.ID, input)
	if err != nil {
		return nil, nil, err
	}
	if !transitionedToPaid {
		commissions, loadErr := s.loadRechargeOrderCommissions(ctx, orderEntity.ID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		_ = tx.Rollback()
		return rechargeOrderEntityToService(orderEntity), commissions, nil
	}

	if _, err := tx.User.UpdateOneID(referredUser.ID).
		SetHasSuccessfulRecharge(true).
		AddBalance(input.CreditedAmount).
		Save(ctx); err != nil {
		return nil, nil, err
	}

	commissions, err := s.settleCommissionTx(ctx, tx, referredUser, orderEntity, input.Amount, hadSuccessfulRechargeBefore)
	if err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	s.invalidateBalanceCaches(ctx, referredUser.ID)

	return rechargeOrderEntityToService(orderEntity), commissions, nil
}

func (s *ReferralService) settleCommissionTx(
	ctx context.Context,
	tx *dbent.Tx,
	referredUser *dbent.User,
	order *dbent.RechargeOrder,
	commissionBase float64,
	hadSuccessfulRechargeBefore bool,
) ([]ReferralCommission, error) {
	if referredUser == nil || referredUser.InviterID == nil || *referredUser.InviterID <= 0 {
		return nil, nil
	}
	if referredUser.ID == *referredUser.InviterID {
		return nil, nil
	}
	if s.settingService == nil || !s.settingService.IsAffiliateEnabled(ctx) {
		return nil, nil
	}
	if normalizeReferralCurrency(order.Currency) != ReferralDefaultCurrency {
		return nil, nil
	}

	commissionType := ""
	if !hadSuccessfulRechargeBefore {
		if s.settingService.IsFirstCommissionEnabled(ctx) {
			commissionType = ReferralCommissionTypeFirst
		}
	} else if s.settingService.IsRecurringCommissionEnabled(ctx) {
		commissionType = ReferralCommissionTypeRecurring
	}
	if commissionType == "" {
		return nil, nil
	}

	promoter, err := tx.User.Query().
		Where(dbuser.IDEQ(*referredUser.InviterID), dbuser.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if promoter.Status != StatusActive {
		return nil, nil
	}
	if commissionType == ReferralCommissionTypeRecurring && !promoter.RecurringCommissionEnabled {
		return nil, nil
	}

	rate := s.resolveCommissionRate(ctx, promoter, commissionType)
	if rate <= 0 {
		return nil, nil
	}

	commissionAmount := roundMoney(commissionBase * rate / 100)
	if commissionAmount <= 0 {
		return nil, nil
	}

	if promoter.ReferralWithdrawalDebt > 0 {
		offset := math.Min(commissionAmount, promoter.ReferralWithdrawalDebt)
		offset = roundMoney(offset)
		if offset > 0 {
			commissionAmount = roundMoney(commissionAmount - offset)
			newDebt := roundMoney(promoter.ReferralWithdrawalDebt - offset)
			if newDebt < 0 {
				newDebt = 0
			}
			if _, err := tx.User.UpdateOneID(promoter.ID).SetReferralWithdrawalDebt(newDebt).Save(ctx); err != nil {
				return nil, err
			}
		}
	}
	if commissionAmount <= 0 {
		return nil, nil
	}

	entity, err := tx.ReferralCommission.Create().
		SetPromoterUserID(promoter.ID).
		SetReferredUserID(referredUser.ID).
		SetRechargeOrderID(order.ID).
		SetCommissionType(commissionType).
		SetStatus(ReferralCommissionStatusRecorded).
		SetSourceAmount(commissionBase).
		SetRateSnapshot(rate).
		SetCommissionAmount(commissionAmount).
		SetCurrency(ReferralDefaultCurrency).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return []ReferralCommission{referralCommissionEntityToService(entity)}, nil
}

func (s *ReferralService) resolveCommissionRate(ctx context.Context, promoter *dbent.User, commissionType string) float64 {
	if promoter == nil {
		return 0
	}
	switch commissionType {
	case ReferralCommissionTypeFirst:
		if promoter.CustomFirstCommissionRate != nil {
			return normalizeCommissionRate(*promoter.CustomFirstCommissionRate)
		}
		if s.settingService != nil {
			return s.settingService.GetDefaultFirstCommissionRate(ctx)
		}
	case ReferralCommissionTypeRecurring:
		if promoter.CustomRecurringCommissionRate != nil {
			return normalizeCommissionRate(*promoter.CustomRecurringCommissionRate)
		}
		if s.settingService != nil {
			return s.settingService.GetDefaultRecurringCommissionRate(ctx)
		}
	}
	return 0
}

func (s *ReferralService) upsertRechargeOrderTx(ctx context.Context, tx *dbent.Tx, input *RecordPaidRechargeInput) (*dbent.RechargeOrder, error) {
	if existing, err := tx.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ(input.OrderNo)).
		Only(ctx); err == nil {
		if existing.UserID != input.UserID {
			return nil, ErrRechargeOrderUserMismatch
		}
		if existing.Status == RechargeOrderStatusPaid {
			return existing, nil
		}
		if existing.Status == RechargeOrderStatusRefunded {
			return existing, nil
		}
		return existing.Update().
			SetChannel(input.Channel).
			SetCurrency(input.Currency).
			SetAmount(input.Amount).
			SetCreditedAmount(input.CreditedAmount).
			SetCallbackIdempotencyKey(input.CallbackIdempotencyKey).
			SetNillableCallbackRaw(nilIfBlank(input.CallbackRaw)).
			SetNillableNotes(nilIfBlank(input.Notes)).
			Save(ctx)
	} else if err != nil && !dbent.IsNotFound(err) {
		return nil, err
	}

	if input.ExternalOrderID != "" {
		existing, err := tx.RechargeOrder.Query().
			Where(
				rechargeorder.ChannelEQ(input.Channel),
				rechargeorder.ExternalOrderIDEQ(input.ExternalOrderID),
			).
			Only(ctx)
		if err == nil {
			if existing.UserID != input.UserID {
				return nil, ErrRechargeOrderUserMismatch
			}
			return existing, nil
		}
		if err != nil && !dbent.IsNotFound(err) {
			return nil, err
		}
	}

	builder := tx.RechargeOrder.Create().
		SetUserID(input.UserID).
		SetOrderNo(input.OrderNo).
		SetChannel(input.Channel).
		SetSource("payment").
		SetCurrency(input.Currency).
		SetAmount(input.Amount).
		SetCreditedAmount(input.CreditedAmount).
		SetStatus(RechargeOrderStatusPending).
		SetCallbackIdempotencyKey(input.CallbackIdempotencyKey).
		SetNillableCallbackRaw(nilIfBlank(input.CallbackRaw)).
		SetNillableNotes(nilIfBlank(input.Notes))
	if input.ExternalOrderID != "" {
		builder.SetExternalOrderID(input.ExternalOrderID)
	}
	return builder.Save(ctx)
}

func (s *ReferralService) markRechargeOrderPaidTx(ctx context.Context, tx *dbent.Tx, orderID int64, input *RecordPaidRechargeInput) (*dbent.RechargeOrder, bool, error) {
	if tx == nil || orderID <= 0 || input == nil {
		return nil, false, ErrRechargeOrderInvalid
	}

	now := time.Now()
	affected, err := tx.RechargeOrder.Update().
		Where(
			rechargeorder.IDEQ(orderID),
			rechargeorder.StatusNEQ(RechargeOrderStatusPaid),
			rechargeorder.StatusNEQ(RechargeOrderStatusRefunded),
		).
		SetNillableExternalOrderID(nilIfBlank(input.ExternalOrderID)).
		SetChannel(input.Channel).
		SetCurrency(input.Currency).
		SetAmount(input.Amount).
		SetCreditedAmount(input.CreditedAmount).
		SetStatus(RechargeOrderStatusPaid).
		SetPaidAt(now).
		SetSource("payment").
		SetCallbackIdempotencyKey(input.CallbackIdempotencyKey).
		SetNillableCallbackRaw(nilIfBlank(input.CallbackRaw)).
		SetNillableNotes(nilIfBlank(input.Notes)).
		Save(ctx)
	if err != nil {
		return nil, false, err
	}

	orderEntity, err := tx.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(orderID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, false, ErrRechargeOrderNotFound
		}
		return nil, false, err
	}

	if affected == 0 {
		if orderEntity.Status == RechargeOrderStatusPaid {
			return orderEntity, false, nil
		}
		if orderEntity.Status == RechargeOrderStatusRefunded {
			return nil, false, ErrRechargeOrderStateInvalid
		}
		return nil, false, ErrRechargeOrderStateInvalid
	}
	return orderEntity, true, nil
}

func (s *ReferralService) findExistingPaidRecharge(ctx context.Context, orderNo, channel, externalOrderID string) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}

	var entity *dbent.RechargeOrder
	var err error
	if orderNo != "" {
		entity, err = s.entClient.RechargeOrder.Query().
			Where(rechargeorder.OrderNoEQ(orderNo)).
			Only(ctx)
		if err != nil && !dbent.IsNotFound(err) {
			return nil, nil, err
		}
	}
	if entity == nil && externalOrderID != "" {
		entity, err = s.entClient.RechargeOrder.Query().
			Where(
				rechargeorder.ChannelEQ(channel),
				rechargeorder.ExternalOrderIDEQ(externalOrderID),
			).
			Only(ctx)
		if err != nil && !dbent.IsNotFound(err) {
			return nil, nil, err
		}
	}
	if entity == nil || entity.Status != RechargeOrderStatusPaid {
		return nil, nil, nil
	}

	commissionEntities, err := s.entClient.ReferralCommission.Query().
		Where(referralcommission.RechargeOrderIDEQ(entity.ID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}
	commissions := make([]ReferralCommission, 0, len(commissionEntities))
	for _, item := range commissionEntities {
		commissions = append(commissions, referralCommissionEntityToService(item))
	}
	return rechargeOrderEntityToService(entity), commissions, nil
}

func (s *ReferralService) invalidateBalanceCaches(ctx context.Context, userID int64) {
	if userID <= 0 {
		return
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	if s.billingCacheService != nil {
		go func() {
			cacheCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.billingCacheService.InvalidateUserBalance(cacheCtx, userID)
		}()
	}
}

func rechargeOrderEntityToService(entity *dbent.RechargeOrder) *RechargeOrder {
	if entity == nil {
		return nil
	}
	return &RechargeOrder{
		ID:                     entity.ID,
		UserID:                 entity.UserID,
		OrderNo:                entity.OrderNo,
		ExternalOrderID:        entity.ExternalOrderID,
		Channel:                entity.Channel,
		Source:                 entity.Source,
		Currency:               entity.Currency,
		Amount:                 entity.Amount,
		CreditedAmount:         entity.CreditedAmount,
		Status:                 entity.Status,
		PaidAt:                 entity.PaidAt,
		RefundedAt:             entity.RefundedAt,
		CallbackIdempotencyKey: entity.CallbackIdempotencyKey,
		CallbackRaw:            entity.CallbackRaw,
		Notes:                  entity.Notes,
		CreatedAt:              entity.CreatedAt,
		UpdatedAt:              entity.UpdatedAt,
	}
}

func entUserToService(entity *dbent.User) *User {
	if entity == nil {
		return nil
	}
	return &User{
		ID:                            entity.ID,
		Email:                         entity.Email,
		Username:                      entity.Username,
		Notes:                         entity.Notes,
		PasswordHash:                  entity.PasswordHash,
		Role:                          entity.Role,
		Balance:                       entity.Balance,
		Concurrency:                   entity.Concurrency,
		Status:                        entity.Status,
		InviterID:                     entity.InviterID,
		ReferralCode:                  entity.ReferralCode,
		CustomFirstCommissionRate:     entity.CustomFirstCommissionRate,
		CustomRecurringCommissionRate: entity.CustomRecurringCommissionRate,
		RecurringCommissionEnabled:    entity.RecurringCommissionEnabled,
		SoraStorageQuotaBytes:         entity.SoraStorageQuotaBytes,
		SoraStorageUsedBytes:          entity.SoraStorageUsedBytes,
		TotpSecretEncrypted:           entity.TotpSecretEncrypted,
		TotpEnabled:                   entity.TotpEnabled,
		TotpEnabledAt:                 entity.TotpEnabledAt,
		CreatedAt:                     entity.CreatedAt,
		UpdatedAt:                     entity.UpdatedAt,
	}
}

func referralCommissionEntityToService(entity *dbent.ReferralCommission) ReferralCommission {
	return ReferralCommission{
		ID:               entity.ID,
		PromoterUserID:   entity.PromoterUserID,
		ReferredUserID:   entity.ReferredUserID,
		RechargeOrderID:  entity.RechargeOrderID,
		CommissionType:   entity.CommissionType,
		Status:           entity.Status,
		SourceAmount:     entity.SourceAmount,
		RateSnapshot:     entity.RateSnapshot,
		CommissionAmount: entity.CommissionAmount,
		Currency:         normalizeReferralCurrency(entity.Currency),
		ReversedAt:       entity.ReversedAt,
		ReversedReason:   entity.ReversedReason,
		Notes:            entity.Notes,
		CreatedAt:        entity.CreatedAt,
		UpdatedAt:        entity.UpdatedAt,
	}
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	switch {
	case pageSize <= 0:
		pageSize = 20
	case pageSize > 100:
		pageSize = 100
	}
	return page, pageSize
}

func roundMoney(value float64) float64 {
	return math.Round(value*1e8) / 1e8
}

func nilIfBlank(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func derefOrValue(value string, current *string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	if current != nil {
		return *current
	}
	return ""
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func referralCommissionUnlockAt(createdAt time.Time) time.Time {
	return createdAt.AddDate(0, 1, 0)
}

func normalizeReferralCurrency(currency string) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return ReferralDefaultCurrency
	}
	return currency
}

func isForUpdateUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "for update/share not supported in sqlite")
}

func (s *ReferralService) DebugString(order *RechargeOrder) string {
	if order == nil {
		return ""
	}
	return fmt.Sprintf("%s:%s:%.4f", order.OrderNo, order.Status, order.Amount)
}
