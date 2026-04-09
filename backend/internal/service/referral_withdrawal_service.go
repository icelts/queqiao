package service

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalrequest"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrReferralWithdrawalInvalid      = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_INVALID", "invalid referral withdrawal request")
	ErrReferralWithdrawalNotFound     = infraerrors.NotFound("REFERRAL_WITHDRAWAL_NOT_FOUND", "referral withdrawal request not found")
	ErrReferralWithdrawalStateInvalid = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_STATE_INVALID", "invalid referral withdrawal request state")
	ErrReferralWithdrawalDisabled     = infraerrors.Forbidden("REFERRAL_WITHDRAWAL_DISABLED", "referral withdrawal is disabled")
	ErrReferralWithdrawalThreshold    = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_THRESHOLD_NOT_MET", "referral withdrawal threshold not met")
	ErrReferralWithdrawalInsufficient = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_INSUFFICIENT", "insufficient withdrawable commission")
)

type CreateReferralWithdrawalInput struct {
	UserID            int64
	Amount            float64
	Currency          string
	PaymentMethod     string
	AccountName       string
	AccountIdentifier string
	Notes             string
}

type ReviewReferralWithdrawalInput struct {
	RequestID      int64
	ReviewerUserID int64
	ReviewNotes    string
}

type referralWithdrawalStats struct {
	EffectiveInviteeCount    int64
	WithdrawEnabled          bool
	WithdrawMinAmount        float64
	WithdrawMinInvitees      int64
	EligibleCommission       float64
	AvailableCommission      float64
	FrozenCommission         float64
	NextUnlockAt             *time.Time
	PendingWithdrawalAmount  float64
	ApprovedWithdrawalAmount float64
	CanWithdraw              bool
}

func (s *ReferralService) getWithdrawalStats(ctx context.Context, promoterUserID int64) (*referralWithdrawalStats, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	return s.buildWithdrawalStats(ctx, promoterUserID, s.entClient.ReferralCommission.Query(), s.entClient.ReferralWithdrawalRequest.Query())
}

func (s *ReferralService) buildWithdrawalStats(
	ctx context.Context,
	promoterUserID int64,
	commissionQuery *dbent.ReferralCommissionQuery,
	withdrawalQuery *dbent.ReferralWithdrawalRequestQuery,
) (*referralWithdrawalStats, error) {
	if promoterUserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}

	stats := &referralWithdrawalStats{}
	if s.settingService != nil {
		stats.WithdrawEnabled = s.settingService.IsAffiliateEnabled(ctx) && s.settingService.IsAffiliateWithdrawEnabled(ctx)
		stats.WithdrawMinAmount = s.settingService.GetAffiliateWithdrawMinAmount(ctx)
		stats.WithdrawMinInvitees = s.settingService.GetAffiliateWithdrawMinInvitees(ctx)
	}

	commissions, err := commissionQuery.
		Where(
			referralcommission.PromoterUserIDEQ(promoterUserID),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	effectiveInvitees := make(map[int64]struct{}, len(commissions))
	eligibleRecorded := 0.0
	frozenCommission := 0.0
	now := time.Now()
	var nextUnlockAt *time.Time
	for _, item := range commissions {
		unlockAt := referralCommissionUnlockAt(item.CreatedAt)
		if !unlockAt.After(now) {
			eligibleRecorded = roundMoney(eligibleRecorded + item.CommissionAmount)
			effectiveInvitees[item.ReferredUserID] = struct{}{}
			continue
		}
		frozenCommission = roundMoney(frozenCommission + item.CommissionAmount)
		if nextUnlockAt == nil || unlockAt.Before(*nextUnlockAt) {
			unlockAtCopy := unlockAt
			nextUnlockAt = &unlockAtCopy
		}
	}
	stats.EffectiveInviteeCount = int64(len(effectiveInvitees))
	stats.EligibleCommission = eligibleRecorded
	stats.FrozenCommission = frozenCommission
	stats.NextUnlockAt = nextUnlockAt

	withdrawals, err := withdrawalQuery.
		Where(referralwithdrawalrequest.PromoterUserIDEQ(promoterUserID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	for _, item := range withdrawals {
		switch item.Status {
		case ReferralWithdrawalStatusPending:
			stats.PendingWithdrawalAmount = roundMoney(stats.PendingWithdrawalAmount + item.Amount)
		case ReferralWithdrawalStatusApproved:
			stats.ApprovedWithdrawalAmount = roundMoney(stats.ApprovedWithdrawalAmount + item.Amount)
		}
	}

	stats.AvailableCommission = roundMoney(eligibleRecorded - stats.PendingWithdrawalAmount - stats.ApprovedWithdrawalAmount)
	if stats.AvailableCommission < 0 {
		stats.AvailableCommission = 0
	}
	stats.CanWithdraw = stats.WithdrawEnabled &&
		stats.AvailableCommission >= stats.WithdrawMinAmount &&
		stats.EffectiveInviteeCount >= stats.WithdrawMinInvitees

	return stats, nil
}

func (s *ReferralService) CreateWithdrawalRequest(ctx context.Context, input *CreateReferralWithdrawalInput) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.UserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}

	input.Amount = roundMoney(input.Amount)
	input.Currency = firstNonBlank(input.Currency, "CNY")
	input.PaymentMethod = firstNonBlank(input.PaymentMethod)
	input.AccountName = firstNonBlank(input.AccountName)
	input.AccountIdentifier = firstNonBlank(input.AccountIdentifier)
	input.Notes = firstNonBlank(input.Notes)
	if input.Amount <= 0 || input.PaymentMethod == "" || input.AccountIdentifier == "" {
		return nil, ErrReferralWithdrawalInvalid
	}

	user, err := s.entClient.User.Query().
		Where(dbuser.IDEQ(input.UserID), dbuser.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if user.Status != StatusActive {
		return nil, ErrUserNotActive
	}

	stats, err := s.getWithdrawalStats(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if !stats.WithdrawEnabled {
		return nil, ErrReferralWithdrawalDisabled
	}
	if stats.EffectiveInviteeCount < stats.WithdrawMinInvitees || input.Amount < stats.WithdrawMinAmount {
		return nil, ErrReferralWithdrawalThreshold
	}
	if input.Amount > stats.AvailableCommission {
		return nil, ErrReferralWithdrawalInsufficient
	}

	entity, err := s.entClient.ReferralWithdrawalRequest.Create().
		SetPromoterUserID(input.UserID).
		SetAmount(input.Amount).
		SetCurrency(input.Currency).
		SetPaymentMethod(input.PaymentMethod).
		SetNillableAccountName(nilIfBlank(input.AccountName)).
		SetAccountIdentifier(input.AccountIdentifier).
		SetStatus(ReferralWithdrawalStatusPending).
		SetNillableNotes(nilIfBlank(input.Notes)).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) CancelWithdrawalRequest(ctx context.Context, userID, requestID int64) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if userID <= 0 || requestID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}

	entity, err := s.entClient.ReferralWithdrawalRequest.Query().
		Where(
			referralwithdrawalrequest.IDEQ(requestID),
			referralwithdrawalrequest.PromoterUserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrReferralWithdrawalNotFound
		}
		return nil, err
	}
	if entity.Status == ReferralWithdrawalStatusCanceled {
		return referralWithdrawalRequestEntityToService(entity), nil
	}
	if entity.Status != ReferralWithdrawalStatusPending {
		return nil, ErrReferralWithdrawalStateInvalid
	}

	entity, err = entity.Update().
		SetStatus(ReferralWithdrawalStatusCanceled).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) ListWithdrawalRequests(ctx context.Context, promoterUserID int64, page, pageSize int) ([]ReferralWithdrawalRequest, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}
	if promoterUserID <= 0 {
		return nil, 0, ErrReferralWithdrawalInvalid
	}

	page, pageSize = normalizePage(page, pageSize)
	baseQuery := s.entClient.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.PromoterUserIDEQ(promoterUserID))

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := baseQuery.
		Order(dbent.Desc(referralwithdrawalrequest.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]ReferralWithdrawalRequest, 0, len(items))
	for _, item := range items {
		result = append(result, *referralWithdrawalRequestEntityToService(item))
	}
	return result, int64(total), nil
}

func (s *ReferralService) ListAdminWithdrawalRequests(ctx context.Context, status string, page, pageSize int) ([]ReferralWithdrawalRequestDetail, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}

	page, pageSize = normalizePage(page, pageSize)
	baseQuery := s.entClient.ReferralWithdrawalRequest.Query()
	status = firstNonBlank(status)
	switch status {
	case ReferralWithdrawalStatusPending, ReferralWithdrawalStatusApproved, ReferralWithdrawalStatusRejected, ReferralWithdrawalStatusCanceled:
		baseQuery = baseQuery.Where(referralwithdrawalrequest.StatusEQ(status))
	}

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := baseQuery.
		Order(dbent.Desc(referralwithdrawalrequest.FieldID)).
		WithPromoter().
		WithReviewer().
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]ReferralWithdrawalRequestDetail, 0, len(items))
	for _, item := range items {
		result = append(result, ReferralWithdrawalRequestDetail{
			Request:  *referralWithdrawalRequestEntityToService(item),
			Promoter: entUserToService(item.Edges.Promoter),
			Reviewer: entUserToService(item.Edges.Reviewer),
		})
	}

	return result, int64(total), nil
}

func (s *ReferralService) ApproveWithdrawalRequest(ctx context.Context, input *ReviewReferralWithdrawalInput) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.RequestID <= 0 || input.ReviewerUserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}
	input.ReviewNotes = firstNonBlank(input.ReviewNotes)

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	entity, err := tx.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.IDEQ(input.RequestID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrReferralWithdrawalNotFound
		}
		return nil, err
	}
	if entity.Status == ReferralWithdrawalStatusApproved {
		return referralWithdrawalRequestEntityToService(entity), nil
	}
	if entity.Status != ReferralWithdrawalStatusPending {
		return nil, ErrReferralWithdrawalStateInvalid
	}

	stats, err := s.buildWithdrawalStats(ctx, entity.PromoterUserID, tx.ReferralCommission.Query(), tx.ReferralWithdrawalRequest.Query())
	if err != nil {
		return nil, err
	}
	if !stats.WithdrawEnabled {
		return nil, ErrReferralWithdrawalDisabled
	}
	if stats.EffectiveInviteeCount < stats.WithdrawMinInvitees {
		return nil, ErrReferralWithdrawalThreshold
	}
	otherPendingAmount := roundMoney(stats.PendingWithdrawalAmount - entity.Amount)
	if otherPendingAmount < 0 {
		otherPendingAmount = 0
	}
	approvalAvailable := roundMoney(stats.EligibleCommission - stats.ApprovedWithdrawalAmount - otherPendingAmount)
	if entity.Amount > approvalAvailable {
		return nil, ErrReferralWithdrawalInsufficient
	}

	now := time.Now()
	entity, err = entity.Update().
		SetStatus(ReferralWithdrawalStatusApproved).
		SetReviewerUserID(input.ReviewerUserID).
		SetReviewedAt(now).
		SetNillableReviewNotes(nilIfBlank(input.ReviewNotes)).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	if entity.Amount != 0 {
		if _, err := tx.User.UpdateOneID(entity.PromoterUserID).AddBalance(-entity.Amount).Save(ctx); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.invalidateBalanceCaches(ctx, entity.PromoterUserID)

	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) RejectWithdrawalRequest(ctx context.Context, input *ReviewReferralWithdrawalInput) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.RequestID <= 0 || input.ReviewerUserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}

	entity, err := s.entClient.ReferralWithdrawalRequest.Query().
		Where(referralwithdrawalrequest.IDEQ(input.RequestID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrReferralWithdrawalNotFound
		}
		return nil, err
	}
	if entity.Status == ReferralWithdrawalStatusRejected {
		return referralWithdrawalRequestEntityToService(entity), nil
	}
	if entity.Status != ReferralWithdrawalStatusPending {
		return nil, ErrReferralWithdrawalStateInvalid
	}

	entity, err = entity.Update().
		SetStatus(ReferralWithdrawalStatusRejected).
		SetReviewerUserID(input.ReviewerUserID).
		SetReviewedAt(time.Now()).
		SetNillableReviewNotes(nilIfBlank(input.ReviewNotes)).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return referralWithdrawalRequestEntityToService(entity), nil
}

func referralWithdrawalRequestEntityToService(entity *dbent.ReferralWithdrawalRequest) *ReferralWithdrawalRequest {
	if entity == nil {
		return nil
	}

	return &ReferralWithdrawalRequest{
		ID:                entity.ID,
		PromoterUserID:    entity.PromoterUserID,
		ReviewerUserID:    entity.ReviewerUserID,
		Amount:            entity.Amount,
		Currency:          entity.Currency,
		PaymentMethod:     entity.PaymentMethod,
		AccountName:       entity.AccountName,
		AccountIdentifier: entity.AccountIdentifier,
		Status:            entity.Status,
		ReviewedAt:        entity.ReviewedAt,
		Notes:             entity.Notes,
		ReviewNotes:       entity.ReviewNotes,
		CreatedAt:         entity.CreatedAt,
		UpdatedAt:         entity.UpdatedAt,
	}
}
