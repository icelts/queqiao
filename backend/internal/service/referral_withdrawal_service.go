package service

import (
	"context"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalallocation"
	"github.com/Wei-Shaw/sub2api/ent/referralwithdrawalrequest"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var (
	ErrReferralWithdrawalInvalid          = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_INVALID", "invalid referral withdrawal request")
	ErrReferralWithdrawalNotFound         = infraerrors.NotFound("REFERRAL_WITHDRAWAL_NOT_FOUND", "referral withdrawal request not found")
	ErrReferralWithdrawalStateInvalid     = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_STATE_INVALID", "invalid referral withdrawal request state")
	ErrReferralWithdrawalDisabled         = infraerrors.Forbidden("REFERRAL_WITHDRAWAL_DISABLED", "referral withdrawal is disabled")
	ErrReferralWithdrawalThreshold        = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_THRESHOLD_NOT_MET", "referral withdrawal threshold not met")
	ErrReferralWithdrawalInsufficient     = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_INSUFFICIENT", "insufficient withdrawable commission")
	ErrReferralWithdrawalCurrencyConflict = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_CURRENCY_CONFLICT", "mixed referral commission currencies require manual reconciliation")
	ErrReferralWithdrawalDebtOutstanding  = infraerrors.BadRequest("REFERRAL_WITHDRAWAL_DEBT_OUTSTANDING", "outstanding referral withdrawal debt must be settled first")
	ErrReferralWithdrawalPendingExists    = infraerrors.Conflict("REFERRAL_WITHDRAWAL_PENDING_EXISTS", "an existing pending withdrawal request must be reviewed first")
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

type MarkReferralWithdrawalPaidInput struct {
	RequestID      int64
	OperatorUserID int64
	PaymentNotes   string
}

type referralWithdrawalStats struct {
	EffectiveInviteeCount    int64
	WithdrawEnabled          bool
	WithdrawMinAmount        float64
	WithdrawMinInvitees      int64
	Currency                 string
	HasMixedCurrencies       bool
	EligibleCommission       float64
	AvailableCommission      float64
	FrozenCommission         float64
	NextUnlockAt             *time.Time
	PendingWithdrawalAmount  float64
	ApprovedWithdrawalAmount float64
	PaidWithdrawalAmount     float64
	WithdrawalDebt           float64
	CanWithdraw              bool
}

func (s *ReferralService) getWithdrawalStats(ctx context.Context, promoterUserID int64) (*referralWithdrawalStats, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	return s.buildWithdrawalStats(
		ctx,
		promoterUserID,
		s.entClient.ReferralCommission.Query(),
		s.entClient.ReferralWithdrawalRequest.Query(),
		s.entClient.ReferralWithdrawalAllocation.Query(),
		s.entClient.User.Query(),
	)
}

func (s *ReferralService) buildWithdrawalStats(
	ctx context.Context,
	promoterUserID int64,
	commissionQuery *dbent.ReferralCommissionQuery,
	withdrawalQuery *dbent.ReferralWithdrawalRequestQuery,
	allocationQuery *dbent.ReferralWithdrawalAllocationQuery,
	userQuery *dbent.UserQuery,
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

	promoter, err := userQuery.
		Where(dbuser.IDEQ(promoterUserID), dbuser.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	stats.WithdrawalDebt = roundMoney(promoter.ReferralWithdrawalDebt)
	if stats.WithdrawalDebt < 0 {
		stats.WithdrawalDebt = 0
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
	eligibleByCurrency := make(map[string]float64)
	frozenByCurrency := make(map[string]float64)
	consumedByCurrency := make(map[string]float64)
	pendingByCurrency := make(map[string]float64)
	approvedByCurrency := make(map[string]float64)
	paidByCurrency := make(map[string]float64)
	currencySet := make(map[string]struct{})
	now := time.Now()
	var nextUnlockAt *time.Time
	maturedCommissionCurrency := make(map[int64]string)
	for _, item := range commissions {
		currency := normalizeReferralCurrency(item.Currency)
		currencySet[currency] = struct{}{}
		unlockAt := referralCommissionUnlockAt(item.CreatedAt)
		if !unlockAt.After(now) {
			maturedCommissionCurrency[item.ID] = currency
			eligibleByCurrency[currency] = roundMoney(eligibleByCurrency[currency] + item.CommissionAmount)
			effectiveInvitees[item.ReferredUserID] = struct{}{}
			continue
		}
		frozenByCurrency[currency] = roundMoney(frozenByCurrency[currency] + item.CommissionAmount)
		if nextUnlockAt == nil || unlockAt.Before(*nextUnlockAt) {
			unlockAtCopy := unlockAt
			nextUnlockAt = &unlockAtCopy
		}
	}
	stats.EffectiveInviteeCount = int64(len(effectiveInvitees))
	stats.NextUnlockAt = nextUnlockAt

	withdrawals, err := withdrawalQuery.
		Where(referralwithdrawalrequest.PromoterUserIDEQ(promoterUserID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range withdrawals {
		currency := normalizeReferralCurrency(item.Currency)
		switch item.Status {
		case ReferralWithdrawalStatusPending:
			pendingByCurrency[currency] = roundMoney(pendingByCurrency[currency] + item.Amount)
			currencySet[currency] = struct{}{}
		case ReferralWithdrawalStatusApproved:
			approvedByCurrency[currency] = roundMoney(approvedByCurrency[currency] + item.Amount)
			currencySet[currency] = struct{}{}
		case ReferralWithdrawalStatusPaid:
			paidByCurrency[currency] = roundMoney(paidByCurrency[currency] + item.Amount)
			currencySet[currency] = struct{}{}
		}
	}

	allocations, err := allocationQuery.
		Where(
			referralwithdrawalallocation.PromoterUserIDEQ(promoterUserID),
			referralwithdrawalallocation.HasWithdrawalRequestWith(
				referralwithdrawalrequest.StatusIn(
					ReferralWithdrawalStatusPending,
					ReferralWithdrawalStatusApproved,
					ReferralWithdrawalStatusPaid,
				),
			),
			referralwithdrawalallocation.HasCommissionWith(
				referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
			),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, item := range allocations {
		currency, ok := maturedCommissionCurrency[item.CommissionID]
		if !ok {
			continue
		}
		consumedByCurrency[currency] = roundMoney(consumedByCurrency[currency] + item.Amount)
	}

	stats.Currency, stats.HasMixedCurrencies = resolveReferralLedgerCurrency(currencySet)
	if stats.HasMixedCurrencies {
		stats.CanWithdraw = false
		return stats, nil
	}

	stats.EligibleCommission = eligibleByCurrency[stats.Currency]
	stats.FrozenCommission = frozenByCurrency[stats.Currency]
	stats.PendingWithdrawalAmount = pendingByCurrency[stats.Currency]
	stats.ApprovedWithdrawalAmount = approvedByCurrency[stats.Currency]
	stats.PaidWithdrawalAmount = paidByCurrency[stats.Currency]
	stats.AvailableCommission = roundMoney(stats.EligibleCommission - consumedByCurrency[stats.Currency] - stats.WithdrawalDebt)
	if stats.AvailableCommission < 0 {
		stats.AvailableCommission = 0
	}
	stats.CanWithdraw = stats.WithdrawEnabled &&
		stats.WithdrawalDebt <= 0 &&
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
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
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

	hasPendingRequest, err := s.entClient.ReferralWithdrawalRequest.Query().
		Where(
			referralwithdrawalrequest.PromoterUserIDEQ(input.UserID),
			referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusPending),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if hasPendingRequest {
		return nil, ErrReferralWithdrawalPendingExists
	}

	stats, err := s.getWithdrawalStats(ctx, input.UserID)
	if err != nil {
		return nil, err
	}
	if !stats.WithdrawEnabled {
		return nil, ErrReferralWithdrawalDisabled
	}
	if stats.HasMixedCurrencies {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	if stats.WithdrawalDebt > 0 {
		return nil, ErrReferralWithdrawalDebtOutstanding
	}
	if stats.EffectiveInviteeCount < stats.WithdrawMinInvitees || input.Amount < stats.WithdrawMinAmount {
		return nil, ErrReferralWithdrawalThreshold
	}
	if input.Amount > stats.AvailableCommission {
		return nil, ErrReferralWithdrawalInsufficient
	}
	if input.Currency != "" && stats.Currency != "" && normalizeReferralCurrency(input.Currency) != stats.Currency {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	input.Currency = firstNonBlank(stats.Currency, normalizeReferralCurrency(input.Currency), ReferralDefaultCurrency)

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	hasPendingRequest, err = tx.ReferralWithdrawalRequest.Query().
		Where(
			referralwithdrawalrequest.PromoterUserIDEQ(input.UserID),
			referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusPending),
		).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if hasPendingRequest {
		return nil, ErrReferralWithdrawalPendingExists
	}

	txStats, err := s.buildWithdrawalStats(
		ctx,
		input.UserID,
		tx.ReferralCommission.Query(),
		tx.ReferralWithdrawalRequest.Query(),
		tx.ReferralWithdrawalAllocation.Query(),
		tx.User.Query(),
	)
	if err != nil {
		return nil, err
	}
	if !txStats.WithdrawEnabled {
		return nil, ErrReferralWithdrawalDisabled
	}
	if txStats.HasMixedCurrencies {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	if txStats.WithdrawalDebt > 0 {
		return nil, ErrReferralWithdrawalDebtOutstanding
	}
	if txStats.EffectiveInviteeCount < txStats.WithdrawMinInvitees || input.Amount < txStats.WithdrawMinAmount {
		return nil, ErrReferralWithdrawalThreshold
	}
	if input.Amount > txStats.AvailableCommission {
		return nil, ErrReferralWithdrawalInsufficient
	}
	if input.Currency != "" && txStats.Currency != "" && normalizeReferralCurrency(input.Currency) != txStats.Currency {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	input.Currency = firstNonBlank(txStats.Currency, normalizeReferralCurrency(input.Currency), ReferralDefaultCurrency)

	entity, err := tx.ReferralWithdrawalRequest.Create().
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
		if dbent.IsConstraintError(err) {
			return nil, ErrReferralWithdrawalPendingExists
		}
		return nil, err
	}

	if err := s.allocateWithdrawalCommissionsTx(ctx, tx, entity); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
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

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	entity, err := tx.ReferralWithdrawalRequest.Query().
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
		Where(referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusPending)).
		SetStatus(ReferralWithdrawalStatusCanceled).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrReferralWithdrawalStateInvalid
		}
		return nil, err
	}
	if err := s.releaseWithdrawalAllocationsTx(ctx, tx, entity.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
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
	case ReferralWithdrawalStatusPending, ReferralWithdrawalStatusApproved, ReferralWithdrawalStatusPaid, ReferralWithdrawalStatusRejected, ReferralWithdrawalStatusCanceled:
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

	stats, err := s.buildWithdrawalStats(
		ctx,
		entity.PromoterUserID,
		tx.ReferralCommission.Query(),
		tx.ReferralWithdrawalRequest.Query(),
		tx.ReferralWithdrawalAllocation.Query(),
		tx.User.Query(),
	)
	if err != nil {
		return nil, err
	}
	if !stats.WithdrawEnabled {
		return nil, ErrReferralWithdrawalDisabled
	}
	if stats.HasMixedCurrencies {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	if stats.WithdrawalDebt > 0 {
		return nil, ErrReferralWithdrawalDebtOutstanding
	}
	if stats.EffectiveInviteeCount < stats.WithdrawMinInvitees {
		return nil, ErrReferralWithdrawalThreshold
	}
	if stats.Currency != "" && normalizeReferralCurrency(entity.Currency) != stats.Currency {
		return nil, ErrReferralWithdrawalCurrencyConflict
	}
	allocatedAmount, err := s.sumRecordedAllocationsForRequestTx(ctx, tx, entity.ID)
	if err != nil {
		return nil, err
	}
	if allocatedAmount+1e-9 < entity.Amount {
		return nil, ErrReferralWithdrawalInsufficient
	}

	now := time.Now()
	entity, err = entity.Update().
		Where(referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusPending)).
		SetStatus(ReferralWithdrawalStatusApproved).
		SetReviewerUserID(input.ReviewerUserID).
		SetReviewedAt(now).
		SetNillableReviewNotes(nilIfBlank(input.ReviewNotes)).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			entity, err = tx.ReferralWithdrawalRequest.Query().
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
			return nil, ErrReferralWithdrawalStateInvalid
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) RejectWithdrawalRequest(ctx context.Context, input *ReviewReferralWithdrawalInput) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.RequestID <= 0 || input.ReviewerUserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}

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
	if entity.Status == ReferralWithdrawalStatusRejected {
		return referralWithdrawalRequestEntityToService(entity), nil
	}
	if entity.Status != ReferralWithdrawalStatusPending && entity.Status != ReferralWithdrawalStatusApproved {
		return nil, ErrReferralWithdrawalStateInvalid
	}

	entity, err = entity.Update().
		Where(referralwithdrawalrequest.StatusIn(ReferralWithdrawalStatusPending, ReferralWithdrawalStatusApproved)).
		SetStatus(ReferralWithdrawalStatusRejected).
		SetReviewerUserID(input.ReviewerUserID).
		SetReviewedAt(time.Now()).
		SetNillableReviewNotes(nilIfBlank(input.ReviewNotes)).
		Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			entity, err = tx.ReferralWithdrawalRequest.Query().
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
			return nil, ErrReferralWithdrawalStateInvalid
		}
		return nil, err
	}
	if err := s.releaseWithdrawalAllocationsTx(ctx, tx, entity.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) MarkWithdrawalRequestPaid(ctx context.Context, input *MarkReferralWithdrawalPaidInput) (*ReferralWithdrawalRequest, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.RequestID <= 0 || input.OperatorUserID <= 0 {
		return nil, ErrReferralWithdrawalInvalid
	}
	input.PaymentNotes = firstNonBlank(input.PaymentNotes)

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
	if entity.Status == ReferralWithdrawalStatusPaid {
		return referralWithdrawalRequestEntityToService(entity), nil
	}
	if entity.Status != ReferralWithdrawalStatusApproved {
		return nil, ErrReferralWithdrawalStateInvalid
	}

	allocatedAmount, err := s.sumRecordedAllocationsForRequestTx(ctx, tx, entity.ID)
	if err != nil {
		return nil, err
	}
	if allocatedAmount+1e-9 < entity.Amount {
		return nil, ErrReferralWithdrawalInsufficient
	}

	updater := entity.Update().
		Where(referralwithdrawalrequest.StatusEQ(ReferralWithdrawalStatusApproved)).
		SetStatus(ReferralWithdrawalStatusPaid).
		SetPaidAt(time.Now()).
		SetNillableReviewNotes(firstNonBlankPtr(nilIfBlank(input.PaymentNotes), entity.ReviewNotes))
	if entity.ReviewerUserID == nil {
		updater.SetReviewerUserID(input.OperatorUserID)
		if entity.ReviewedAt == nil {
			updater.SetReviewedAt(time.Now())
		}
	}
	entity, err = updater.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			entity, err = tx.ReferralWithdrawalRequest.Query().
				Where(referralwithdrawalrequest.IDEQ(input.RequestID)).
				Only(ctx)
			if err != nil {
				if dbent.IsNotFound(err) {
					return nil, ErrReferralWithdrawalNotFound
				}
				return nil, err
			}
			if entity.Status == ReferralWithdrawalStatusPaid {
				return referralWithdrawalRequestEntityToService(entity), nil
			}
			return nil, ErrReferralWithdrawalStateInvalid
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return referralWithdrawalRequestEntityToService(entity), nil
}

func (s *ReferralService) allocateWithdrawalCommissionsTx(ctx context.Context, tx *dbent.Tx, request *dbent.ReferralWithdrawalRequest) error {
	if tx == nil || request == nil || request.ID <= 0 || request.Amount <= 0 {
		return ErrReferralWithdrawalInvalid
	}

	now := time.Now()
	remaining := roundMoney(request.Amount)
	if remaining <= 0 {
		return ErrReferralWithdrawalInvalid
	}
	requestCurrency := normalizeReferralCurrency(request.Currency)

	candidateQuery := tx.ReferralCommission.Query().
		Where(
			referralcommission.PromoterUserIDEQ(request.PromoterUserID),
			referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
			referralcommission.CurrencyEQ(requestCurrency),
		).
		Order(
			dbent.Asc(referralcommission.FieldCreatedAt),
			dbent.Asc(referralcommission.FieldID),
		).
		ForUpdate()
	candidates, err := candidateQuery.All(ctx)
	if err != nil && isForUpdateUnsupportedError(err) {
		candidates, err = tx.ReferralCommission.Query().
			Where(
				referralcommission.PromoterUserIDEQ(request.PromoterUserID),
				referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
				referralcommission.CurrencyEQ(requestCurrency),
			).
			Order(
				dbent.Asc(referralcommission.FieldCreatedAt),
				dbent.Asc(referralcommission.FieldID),
			).
			All(ctx)
	}
	if err != nil {
		return err
	}

	existingAllocations, err := tx.ReferralWithdrawalAllocation.Query().
		Where(
			referralwithdrawalallocation.PromoterUserIDEQ(request.PromoterUserID),
			referralwithdrawalallocation.HasWithdrawalRequestWith(
				referralwithdrawalrequest.StatusIn(
					ReferralWithdrawalStatusPending,
					ReferralWithdrawalStatusApproved,
					ReferralWithdrawalStatusPaid,
				),
			),
			referralwithdrawalallocation.HasCommissionWith(referralcommission.StatusEQ(ReferralCommissionStatusRecorded)),
		).
		All(ctx)
	if err != nil {
		return err
	}
	usedByCommission := make(map[int64]float64)
	for _, item := range existingAllocations {
		usedByCommission[item.CommissionID] = roundMoney(usedByCommission[item.CommissionID] + item.Amount)
	}

	for _, commission := range candidates {
		if remaining <= 0 {
			break
		}
		if referralCommissionUnlockAt(commission.CreatedAt).After(now) {
			continue
		}
		availableInCommission := roundMoney(commission.CommissionAmount - usedByCommission[commission.ID])
		if availableInCommission <= 0 {
			continue
		}

		allocateAmount := roundMoney(math.Min(remaining, availableInCommission))
		if allocateAmount <= 0 {
			continue
		}
		if _, err := tx.ReferralWithdrawalAllocation.Create().
			SetPromoterUserID(request.PromoterUserID).
			SetWithdrawalRequestID(request.ID).
			SetCommissionID(commission.ID).
			SetAmount(allocateAmount).
			Save(ctx); err != nil {
			return err
		}
		usedByCommission[commission.ID] = roundMoney(usedByCommission[commission.ID] + allocateAmount)
		remaining = roundMoney(remaining - allocateAmount)
	}

	if remaining > 0 {
		return ErrReferralWithdrawalInsufficient
	}
	return nil
}

func (s *ReferralService) releaseWithdrawalAllocationsTx(ctx context.Context, tx *dbent.Tx, requestID int64) error {
	if tx == nil || requestID <= 0 {
		return ErrReferralWithdrawalInvalid
	}
	if _, err := tx.ReferralWithdrawalAllocation.Delete().
		Where(referralwithdrawalallocation.WithdrawalRequestIDEQ(requestID)).
		Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (s *ReferralService) sumRecordedAllocationsForRequestTx(ctx context.Context, tx *dbent.Tx, requestID int64) (float64, error) {
	if tx == nil || requestID <= 0 {
		return 0, ErrReferralWithdrawalInvalid
	}
	allocations, err := tx.ReferralWithdrawalAllocation.Query().
		Where(
			referralwithdrawalallocation.WithdrawalRequestIDEQ(requestID),
			referralwithdrawalallocation.HasCommissionWith(
				referralcommission.StatusEQ(ReferralCommissionStatusRecorded),
			),
		).
		All(ctx)
	if err != nil {
		return 0, err
	}
	sum := 0.0
	for _, item := range allocations {
		sum = roundMoney(sum + item.Amount)
	}
	return sum, nil
}

func (s *ReferralService) handleWithdrawalAllocationsOnCommissionReversalTx(
	ctx context.Context,
	tx *dbent.Tx,
	commissionID int64,
	reversedAt time.Time,
	reason string,
) (float64, error) {
	if tx == nil || commissionID <= 0 {
		return 0, nil
	}

	reason = firstNonBlank(reason)
	allocations, err := tx.ReferralWithdrawalAllocation.Query().
		Where(referralwithdrawalallocation.CommissionIDEQ(commissionID)).
		WithWithdrawalRequest().
		All(ctx)
	if err != nil {
		return 0, err
	}
	if len(allocations) == 0 {
		return 0, nil
	}

	needReject := make(map[int64]*dbent.ReferralWithdrawalRequest)
	debtDelta := 0.0
	for _, item := range allocations {
		request := item.Edges.WithdrawalRequest
		if request == nil {
			continue
		}
		switch request.Status {
		case ReferralWithdrawalStatusPending, ReferralWithdrawalStatusApproved:
			needReject[request.ID] = request
		case ReferralWithdrawalStatusPaid:
			debtDelta = roundMoney(debtDelta + item.Amount)
		}
	}

	for requestID, request := range needReject {
		reviewNotes := appendReviewNote(request.ReviewNotes, reason)
		_, err := tx.ReferralWithdrawalRequest.UpdateOneID(requestID).
			Where(referralwithdrawalrequest.StatusIn(ReferralWithdrawalStatusPending, ReferralWithdrawalStatusApproved)).
			SetStatus(ReferralWithdrawalStatusRejected).
			SetReviewedAt(reversedAt).
			SetNillableReviewNotes(reviewNotes).
			Save(ctx)
		if err != nil {
			if dbent.IsNotFound(err) {
				continue
			}
			return 0, err
		}
		if err := s.releaseWithdrawalAllocationsTx(ctx, tx, requestID); err != nil {
			return 0, err
		}
	}

	return roundMoney(debtDelta), nil
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
		Currency:          normalizeReferralCurrency(entity.Currency),
		PaymentMethod:     entity.PaymentMethod,
		AccountName:       entity.AccountName,
		AccountIdentifier: entity.AccountIdentifier,
		Status:            entity.Status,
		ReviewedAt:        entity.ReviewedAt,
		PaidAt:            entity.PaidAt,
		Notes:             entity.Notes,
		ReviewNotes:       entity.ReviewNotes,
		CreatedAt:         entity.CreatedAt,
		UpdatedAt:         entity.UpdatedAt,
	}
}

func resolveReferralLedgerCurrency(currencies map[string]struct{}) (string, bool) {
	switch len(currencies) {
	case 0:
		return ReferralDefaultCurrency, false
	case 1:
		for currency := range currencies {
			return currency, false
		}
	}
	return "", true
}

func appendReviewNote(existing *string, appendText string) *string {
	appendText = strings.TrimSpace(appendText)
	if appendText == "" {
		return existing
	}
	if existing == nil || strings.TrimSpace(*existing) == "" {
		return &appendText
	}
	merged := strings.TrimSpace(*existing) + "\n" + appendText
	return &merged
}
