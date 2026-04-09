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
	ErrRechargeOrderNotFound            = infraerrors.NotFound("RECHARGE_ORDER_NOT_FOUND", "recharge order not found")
	ErrRechargeOrderStateInvalid        = infraerrors.BadRequest("RECHARGE_ORDER_STATE_INVALID", "invalid recharge order state")
	ErrRechargeOrderAmountRequired      = infraerrors.BadRequest("RECHARGE_ORDER_AMOUNT_REQUIRED", "payment amount is required to confirm recharge order")
	ErrRechargeOrderAmountMismatch      = infraerrors.Conflict("RECHARGE_ORDER_AMOUNT_MISMATCH", "paid amount does not match recharge order")
	ErrRechargeOrderCurrencyUnsupported = infraerrors.BadRequest("RECHARGE_ORDER_CURRENCY_UNSUPPORTED", "unsupported recharge currency")
)

const rechargePaidAmountEqualityTolerance = 0.0000001

type CreateRechargeOrderInput struct {
	UserID         int64
	Amount         float64
	CreditedAmount float64
	Channel        string
	Currency       string
	Source         string
	Notes          string
}

type PaymentWebhookInput struct {
	OrderNo                string
	ExternalOrderID        string
	Channel                string
	Status                 string
	Amount                 float64
	CreditedAmount         float64
	Currency               string
	CallbackIdempotencyKey string
	CallbackRaw            string
	Notes                  string
}

type RefundRechargeOrderInput struct {
	OrderNo                string
	ExternalOrderID        string
	Channel                string
	CallbackIdempotencyKey string
	CallbackRaw            string
	Notes                  string
	ReversedReason         string
}

func (s *ReferralService) CreateRechargeOrder(ctx context.Context, input *CreateRechargeOrderInput) (*RechargeOrder, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil || input.UserID <= 0 || input.Amount <= 0 {
		return nil, ErrRechargeOrderInvalid
	}

	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Source = strings.TrimSpace(input.Source)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.Channel == "" {
		input.Channel = "custom"
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.Channel == XunhuPayChannel && input.Currency != "CNY" {
		return nil, ErrRechargeOrderCurrencyUnsupported.WithMetadata(map[string]string{
			"channel":  input.Channel,
			"currency": input.Currency,
		})
	}
	if input.Source == "" {
		input.Source = RechargeOrderSourceBalance
	}
	input.Amount = normalizeRechargeAmountForChannel(input.Channel, input.Currency, input.Amount)
	if input.Amount <= 0 {
		return nil, ErrRechargeOrderInvalid
	}

	creditedAmount := roundMoney(input.CreditedAmount)
	if input.Source == RechargeOrderSourceBalance && creditedAmount <= 0 {
		ratio := 1.0
		if s.settingService != nil {
			ratio = s.settingService.GetBalanceRechargeRatio(ctx)
		}
		creditedAmount = roundMoney(input.Amount * ratio)
	}
	if input.Source != RechargeOrderSourceBalance && creditedAmount < 0 {
		creditedAmount = 0
	}

	userEntity, err := s.entClient.User.Query().
		Where(dbuser.IDEQ(input.UserID), dbuser.DeletedAtIsNil()).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	if userEntity.Status != StatusActive {
		return nil, ErrUserNotActive
	}

	orderNo, err := generateRechargeOrderNo()
	if err != nil {
		return nil, err
	}

	orderEntity, err := s.entClient.RechargeOrder.Create().
		SetUserID(input.UserID).
		SetOrderNo(orderNo).
		SetChannel(input.Channel).
		SetSource(input.Source).
		SetCurrency(input.Currency).
		SetAmount(input.Amount).
		SetCreditedAmount(creditedAmount).
		SetStatus(RechargeOrderStatusPending).
		SetNillableNotes(nilIfBlank(input.Notes)).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	return rechargeOrderEntityToService(orderEntity), nil
}

func (s *ReferralService) GetRechargeOrder(ctx context.Context, userID int64, orderNo string) (*RechargeOrder, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if userID <= 0 || strings.TrimSpace(orderNo) == "" {
		return nil, ErrRechargeOrderInvalid
	}

	entity, err := s.entClient.RechargeOrder.Query().
		Where(
			rechargeorder.UserIDEQ(userID),
			rechargeorder.OrderNoEQ(strings.TrimSpace(orderNo)),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrRechargeOrderNotFound
		}
		return nil, err
	}
	return rechargeOrderEntityToService(entity), nil
}

func (s *ReferralService) SetRechargeOrderExternalOrderID(ctx context.Context, orderNo, channel, externalOrderID string) (*RechargeOrder, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}

	orderNo = strings.TrimSpace(orderNo)
	channel = strings.ToLower(strings.TrimSpace(channel))
	externalOrderID = strings.TrimSpace(externalOrderID)
	if orderNo == "" || externalOrderID == "" {
		return nil, ErrRechargeOrderInvalid
	}

	entity, err := s.entClient.RechargeOrder.Query().
		Where(rechargeorder.OrderNoEQ(orderNo)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrRechargeOrderNotFound
		}
		return nil, err
	}

	currentExternalID := derefString(entity.ExternalOrderID)
	if currentExternalID != "" {
		return rechargeOrderEntityToService(entity), nil
	}

	updated := entity.Update().SetExternalOrderID(externalOrderID)
	if channel != "" {
		updated.SetChannel(channel)
	}
	entity, err = updated.Save(ctx)
	if err != nil {
		return nil, err
	}
	return rechargeOrderEntityToService(entity), nil
}

func (s *ReferralService) ListRechargeOrders(ctx context.Context, userID int64, page, pageSize int) ([]RechargeOrder, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}
	if userID <= 0 {
		return nil, 0, ErrRechargeOrderInvalid
	}

	page, pageSize = normalizePage(page, pageSize)
	baseQuery := s.entClient.RechargeOrder.Query().
		Where(rechargeorder.UserIDEQ(userID))

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	entities, err := baseQuery.
		Order(dbent.Desc(rechargeorder.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]RechargeOrder, 0, len(entities))
	for _, entity := range entities {
		result = append(result, *rechargeOrderEntityToService(entity))
	}
	return result, int64(total), nil
}

func (s *ReferralService) HandlePaymentWebhook(ctx context.Context, input *PaymentWebhookInput) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if input == nil {
		return nil, nil, ErrRechargeOrderInvalid
	}

	input.OrderNo = strings.TrimSpace(input.OrderNo)
	input.ExternalOrderID = strings.TrimSpace(input.ExternalOrderID)
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.CallbackIdempotencyKey = strings.TrimSpace(input.CallbackIdempotencyKey)
	input.CallbackRaw = strings.TrimSpace(input.CallbackRaw)
	input.Notes = strings.TrimSpace(input.Notes)

	if input.OrderNo == "" && input.ExternalOrderID == "" {
		return nil, nil, ErrRechargeOrderInvalid
	}
	switch input.Status {
	case RechargeOrderStatusPaid, RechargeOrderStatusFailed, RechargeOrderStatusRefunded:
	default:
		return nil, nil, ErrRechargeOrderInvalid
	}

	entity, err := s.findRechargeOrderEntity(ctx, input.OrderNo, input.Channel, input.ExternalOrderID)
	if err != nil {
		return nil, nil, err
	}
	if entity == nil {
		return nil, nil, ErrRechargeOrderNotFound
	}
	if strings.TrimSpace(entity.Channel) != "" {
		input.Channel = strings.ToLower(strings.TrimSpace(entity.Channel))
	}
	if strings.TrimSpace(entity.Currency) != "" {
		input.Currency = strings.ToUpper(strings.TrimSpace(entity.Currency))
	}
	if input.Status == RechargeOrderStatusPaid {
		validatedAmount, err := validatePaidRechargeAmount(entity, input.Amount)
		if err != nil {
			return nil, nil, err
		}
		input.Amount = validatedAmount
	}

	switch input.Status {
	case RechargeOrderStatusPaid:
		if entity.Source == RechargeOrderSourceSubscriptionPurchase {
			return s.completeSubscriptionPurchaseOrder(ctx, entity, input)
		}
		amount := normalizeRechargeAmountForChannel(entity.Channel, entity.Currency, input.Amount)
		creditedAmount := roundMoney(input.CreditedAmount)
		if creditedAmount <= 0 {
			if entity.CreditedAmount > 0 {
				creditedAmount = roundMoney(entity.CreditedAmount)
			} else {
				creditedAmount = amount
			}
		}
		currency := input.Currency
		if currency == "" {
			currency = strings.ToUpper(strings.TrimSpace(entity.Currency))
		}
		channel := input.Channel
		if channel == "" {
			channel = strings.ToLower(strings.TrimSpace(entity.Channel))
		}
		return s.RecordPaidRecharge(ctx, &RecordPaidRechargeInput{
			UserID:                 entity.UserID,
			OrderNo:                entity.OrderNo,
			ExternalOrderID:        firstNonBlank(input.ExternalOrderID, derefString(entity.ExternalOrderID)),
			Channel:                channel,
			Amount:                 amount,
			CreditedAmount:         creditedAmount,
			Currency:               currency,
			CallbackIdempotencyKey: input.CallbackIdempotencyKey,
			CallbackRaw:            input.CallbackRaw,
			Notes:                  input.Notes,
		})
	case RechargeOrderStatusFailed:
		order, err := s.MarkRechargeOrderFailed(ctx, input.OrderNo, input.Channel, input.ExternalOrderID, input.CallbackIdempotencyKey, input.CallbackRaw, input.Notes)
		return order, nil, err
	case RechargeOrderStatusRefunded:
		if entity.Source == RechargeOrderSourceSubscriptionPurchase {
			return s.refundSubscriptionPurchaseOrder(ctx, entity, input)
		}
		return s.RefundRechargeOrder(ctx, &RefundRechargeOrderInput{
			OrderNo:                input.OrderNo,
			ExternalOrderID:        input.ExternalOrderID,
			Channel:                input.Channel,
			CallbackIdempotencyKey: input.CallbackIdempotencyKey,
			CallbackRaw:            input.CallbackRaw,
			Notes:                  input.Notes,
			ReversedReason:         "payment_refunded",
		})
	default:
		return nil, nil, ErrRechargeOrderInvalid
	}
}

func validatePaidRechargeAmount(entity *dbent.RechargeOrder, amount float64) (float64, error) {
	if entity == nil {
		return 0, ErrRechargeOrderInvalid
	}

	externalOrderID := ""
	if entity.ExternalOrderID != nil {
		externalOrderID = strings.TrimSpace(*entity.ExternalOrderID)
	}

	paidAmount := normalizeRechargeAmountForChannel(entity.Channel, entity.Currency, amount)
	if paidAmount <= 0 {
		return 0, ErrRechargeOrderAmountRequired
	}

	expectedAmount := normalizeRechargeAmountForChannel(entity.Channel, entity.Currency, entity.Amount)
	if math.Abs(paidAmount-expectedAmount) > rechargePaidAmountEqualityTolerance {
		return 0, ErrRechargeOrderAmountMismatch.WithMetadata(map[string]string{
			"order_no":          strings.TrimSpace(entity.OrderNo),
			"expected_amount":   fmt.Sprintf("%.8f", expectedAmount),
			"received_amount":   fmt.Sprintf("%.8f", paidAmount),
			"external_order_id": externalOrderID,
		})
	}

	return paidAmount, nil
}

func normalizeRechargeAmountForChannel(channel, currency string, amount float64) float64 {
	normalized := roundMoney(amount)
	channel = strings.ToLower(strings.TrimSpace(channel))
	currency = strings.ToUpper(strings.TrimSpace(currency))

	switch {
	case channel == XunhuPayChannel && (currency == "" || currency == "CNY"):
		// XunhuPay currently charges CNY with 2-decimal precision.
		return math.Round(normalized*100) / 100
	default:
		return normalized
	}
}

func (s *ReferralService) MarkRechargeOrderFailed(ctx context.Context, orderNo, channel, externalOrderID, callbackID, callbackRaw, notes string) (*RechargeOrder, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}

	orderNo = strings.TrimSpace(orderNo)
	channel = strings.ToLower(strings.TrimSpace(channel))
	externalOrderID = strings.TrimSpace(externalOrderID)
	callbackID = strings.TrimSpace(callbackID)
	callbackRaw = strings.TrimSpace(callbackRaw)
	notes = strings.TrimSpace(notes)

	entity, err := s.findRechargeOrderEntity(ctx, orderNo, channel, externalOrderID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, ErrRechargeOrderNotFound
	}
	switch entity.Status {
	case RechargeOrderStatusFailed:
		return rechargeOrderEntityToService(entity), nil
	case RechargeOrderStatusPaid, RechargeOrderStatusRefunded:
		return nil, ErrRechargeOrderStateInvalid
	}

	updated, _, err := s.markRechargeOrderFailedOnPending(ctx, entity, callbackID, callbackRaw, notes)
	if err != nil {
		return nil, err
	}
	return rechargeOrderEntityToService(updated), nil
}

func (s *ReferralService) RefundRechargeOrder(ctx context.Context, input *RefundRechargeOrderInput) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if input == nil {
		return nil, nil, ErrRechargeOrderInvalid
	}

	input.OrderNo = strings.TrimSpace(input.OrderNo)
	input.ExternalOrderID = strings.TrimSpace(input.ExternalOrderID)
	input.Channel = strings.ToLower(strings.TrimSpace(input.Channel))
	input.CallbackIdempotencyKey = strings.TrimSpace(input.CallbackIdempotencyKey)
	input.CallbackRaw = strings.TrimSpace(input.CallbackRaw)
	input.Notes = strings.TrimSpace(input.Notes)
	input.ReversedReason = strings.TrimSpace(input.ReversedReason)
	if input.ReversedReason == "" {
		input.ReversedReason = "payment_refunded"
	}

	entity, err := s.findRechargeOrderEntity(ctx, input.OrderNo, input.Channel, input.ExternalOrderID)
	if err != nil {
		return nil, nil, err
	}
	if entity == nil {
		return nil, nil, ErrRechargeOrderNotFound
	}
	if entity.Status == RechargeOrderStatusRefunded {
		commissions, loadErr := s.loadRechargeOrderCommissions(ctx, entity.ID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		return rechargeOrderEntityToService(entity), commissions, nil
	}
	if entity.Status != RechargeOrderStatusPaid {
		return nil, nil, ErrRechargeOrderStateInvalid
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	orderEntity, err := tx.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(entity.ID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil, ErrRechargeOrderNotFound
		}
		return nil, nil, err
	}
	orderEntity, transitionedToRefunded, err := s.markRechargeOrderRefundedTx(ctx, tx, orderEntity, input)
	if err != nil {
		return nil, nil, err
	}
	if !transitionedToRefunded {
		commissions, loadErr := s.loadRechargeOrderCommissions(ctx, orderEntity.ID)
		if loadErr != nil {
			return nil, nil, loadErr
		}
		_ = tx.Rollback()
		return rechargeOrderEntityToService(orderEntity), commissions, nil
	}

	refundAmount := roundMoney(orderEntity.CreditedAmount)
	if refundAmount == 0 {
		refundAmount = roundMoney(orderEntity.Amount)
	}
	if refundAmount != 0 {
		if _, err := tx.User.UpdateOneID(orderEntity.UserID).AddBalance(-refundAmount).Save(ctx); err != nil {
			return nil, nil, err
		}
	}

	commissionEntities, err := tx.ReferralCommission.Query().
		Where(referralcommission.RechargeOrderIDEQ(orderEntity.ID)).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	reversed := make([]ReferralCommission, 0, len(commissionEntities))
	reversedAt := time.Now()
	if orderEntity.RefundedAt != nil {
		reversedAt = *orderEntity.RefundedAt
	}
	debtByPromoter := make(map[int64]float64)
	autoRejectReason := "auto rejected because related recharge was refunded"
	if strings.TrimSpace(orderEntity.OrderNo) != "" {
		autoRejectReason = "auto rejected because recharge order " + strings.TrimSpace(orderEntity.OrderNo) + " was refunded"
	}
	for _, item := range commissionEntities {
		if item.Status != ReferralCommissionStatusRecorded {
			reversed = append(reversed, referralCommissionEntityToService(item))
			continue
		}

		item, err = item.Update().
			SetStatus(ReferralCommissionStatusReversed).
			SetReversedAt(reversedAt).
			SetReversedReason(input.ReversedReason).
			SetNillableNotes(firstNonBlankPtr(nilIfBlank(input.Notes), item.Notes)).
			Save(ctx)
		if err != nil {
			return nil, nil, err
		}
		debtDelta, err := s.handleWithdrawalAllocationsOnCommissionReversalTx(ctx, tx, item.ID, reversedAt, autoRejectReason)
		if err != nil {
			return nil, nil, err
		}
		if debtDelta > 0 {
			debtByPromoter[item.PromoterUserID] = roundMoney(debtByPromoter[item.PromoterUserID] + debtDelta)
		}
		reversed = append(reversed, referralCommissionEntityToService(item))
	}
	for promoterUserID, debtDelta := range debtByPromoter {
		if promoterUserID <= 0 || debtDelta <= 0 {
			continue
		}
		if _, err := tx.User.UpdateOneID(promoterUserID).
			AddReferralWithdrawalDebt(debtDelta).
			Save(ctx); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	s.invalidateBalanceCaches(ctx, orderEntity.UserID)

	return rechargeOrderEntityToService(orderEntity), reversed, nil
}

func (s *ReferralService) findRechargeOrderEntity(ctx context.Context, orderNo, channel, externalOrderID string) (*dbent.RechargeOrder, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}

	if orderNo != "" {
		entity, err := s.entClient.RechargeOrder.Query().
			Where(rechargeorder.OrderNoEQ(orderNo)).
			Only(ctx)
		if err == nil {
			return entity, nil
		}
		if err != nil && !dbent.IsNotFound(err) {
			return nil, err
		}
	}
	if externalOrderID != "" {
		if channel == "" {
			return nil, ErrRechargeOrderInvalid
		}
		entity, err := s.entClient.RechargeOrder.Query().
			Where(
				rechargeorder.ChannelEQ(channel),
				rechargeorder.ExternalOrderIDEQ(externalOrderID),
			).
			Only(ctx)
		if err == nil {
			return entity, nil
		}
		if err != nil && !dbent.IsNotFound(err) {
			return nil, err
		}
	}
	return nil, nil
}

func (s *ReferralService) loadRechargeOrderCommissions(ctx context.Context, orderID int64) ([]ReferralCommission, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if orderID <= 0 {
		return nil, nil
	}

	entities, err := s.entClient.ReferralCommission.Query().
		Where(referralcommission.RechargeOrderIDEQ(orderID)).
		Order(dbent.Asc(referralcommission.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ReferralCommission, 0, len(entities))
	for _, entity := range entities {
		result = append(result, referralCommissionEntityToService(entity))
	}
	return result, nil
}

func (s *ReferralService) markRechargeOrderRefundedTx(ctx context.Context, tx *dbent.Tx, order *dbent.RechargeOrder, input *RefundRechargeOrderInput) (*dbent.RechargeOrder, bool, error) {
	if tx == nil || order == nil || order.ID <= 0 || input == nil {
		return nil, false, ErrRechargeOrderInvalid
	}

	now := time.Now()
	affected, err := tx.RechargeOrder.Update().
		Where(
			rechargeorder.IDEQ(order.ID),
			rechargeorder.StatusEQ(RechargeOrderStatusPaid),
		).
		SetStatus(RechargeOrderStatusRefunded).
		SetRefundedAt(now).
		SetCallbackIdempotencyKey(firstNonBlank(input.CallbackIdempotencyKey, order.CallbackIdempotencyKey)).
		SetNillableCallbackRaw(firstNonBlankPtr(nilIfBlank(input.CallbackRaw), order.CallbackRaw)).
		SetNillableNotes(firstNonBlankPtr(nilIfBlank(input.Notes), order.Notes)).
		Save(ctx)
	if err != nil {
		return nil, false, err
	}

	orderEntity, err := tx.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(order.ID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, false, ErrRechargeOrderNotFound
		}
		return nil, false, err
	}

	if affected == 0 {
		if orderEntity.Status == RechargeOrderStatusRefunded {
			return orderEntity, false, nil
		}
		return nil, false, ErrRechargeOrderStateInvalid
	}
	return orderEntity, true, nil
}

func (s *ReferralService) markRechargeOrderFailedOnPending(ctx context.Context, order *dbent.RechargeOrder, callbackID, callbackRaw, notes string) (*dbent.RechargeOrder, bool, error) {
	if order == nil || order.ID <= 0 {
		return nil, false, ErrRechargeOrderInvalid
	}

	affected, err := s.entClient.RechargeOrder.Update().
		Where(
			rechargeorder.IDEQ(order.ID),
			rechargeorder.StatusEQ(RechargeOrderStatusPending),
		).
		SetStatus(RechargeOrderStatusFailed).
		SetCallbackIdempotencyKey(firstNonBlank(callbackID, order.CallbackIdempotencyKey)).
		SetNillableCallbackRaw(firstNonBlankPtr(nilIfBlank(callbackRaw), order.CallbackRaw)).
		SetNillableNotes(firstNonBlankPtr(nilIfBlank(notes), order.Notes)).
		Save(ctx)
	if err != nil {
		return nil, false, err
	}

	orderEntity, err := s.entClient.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(order.ID)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, false, ErrRechargeOrderNotFound
		}
		return nil, false, err
	}

	if affected == 0 {
		if orderEntity.Status == RechargeOrderStatusFailed {
			return orderEntity, false, nil
		}
		if orderEntity.Status == RechargeOrderStatusPaid || orderEntity.Status == RechargeOrderStatusRefunded {
			return nil, false, ErrRechargeOrderStateInvalid
		}
		return nil, false, ErrRechargeOrderStateInvalid
	}

	return orderEntity, true, nil
}

func (s *ReferralService) completeSubscriptionPurchaseOrder(ctx context.Context, entity *dbent.RechargeOrder, input *PaymentWebhookInput) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if entity == nil || input == nil {
		return nil, nil, ErrRechargeOrderInvalid
	}
	if s.subscriptionService == nil {
		return nil, nil, ErrServiceUnavailable
	}

	meta, err := ParseSubscriptionPurchaseMetadata(derefString(entity.Notes))
	if err != nil {
		return nil, nil, err
	}

	amount := roundMoney(input.Amount)
	if amount <= 0 {
		amount = roundMoney(entity.Amount)
	}
	currency := input.Currency
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(entity.Currency))
	}
	channel := input.Channel
	if channel == "" {
		channel = strings.ToLower(strings.TrimSpace(entity.Channel))
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	orderEntity, err := tx.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(entity.ID)).
		Only(txCtx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil, ErrRechargeOrderNotFound
		}
		return nil, nil, err
	}

	orderEntity, transitionedToPaid, err := s.markRechargeOrderPaidTx(txCtx, tx, orderEntity.ID, &RecordPaidRechargeInput{
		UserID:                 entity.UserID,
		OrderNo:                entity.OrderNo,
		ExternalOrderID:        firstNonBlank(input.ExternalOrderID, derefString(entity.ExternalOrderID)),
		Channel:                channel,
		Amount:                 amount,
		CreditedAmount:         0,
		Currency:               currency,
		CallbackIdempotencyKey: input.CallbackIdempotencyKey,
		CallbackRaw:            input.CallbackRaw,
		Notes:                  derefString(entity.Notes),
	})
	if err != nil {
		return nil, nil, err
	}
	if !transitionedToPaid {
		_ = tx.Rollback()
		return rechargeOrderEntityToService(orderEntity), nil, nil
	}

	if _, _, err := s.subscriptionService.AssignOrExtendSubscription(txCtx, &AssignSubscriptionInput{
		UserID:       entity.UserID,
		GroupID:      meta.GroupID,
		ValidityDays: meta.ValidityDays,
		Notes:        "subscription purchase order " + entity.OrderNo,
	}); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return rechargeOrderEntityToService(orderEntity), nil, nil
}

func (s *ReferralService) refundSubscriptionPurchaseOrder(ctx context.Context, entity *dbent.RechargeOrder, input *PaymentWebhookInput) (*RechargeOrder, []ReferralCommission, error) {
	if s.entClient == nil {
		return nil, nil, ErrServiceUnavailable
	}
	if entity == nil || input == nil {
		return nil, nil, ErrRechargeOrderInvalid
	}
	if s.subscriptionService == nil {
		return nil, nil, ErrServiceUnavailable
	}

	meta, err := ParseSubscriptionPurchaseMetadata(derefString(entity.Notes))
	if err != nil {
		return nil, nil, err
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	orderEntity, err := tx.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(entity.ID)).
		Only(txCtx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil, ErrRechargeOrderNotFound
		}
		return nil, nil, err
	}

	orderEntity, transitionedToRefunded, err := s.markRechargeOrderRefundedTx(txCtx, tx, orderEntity, &RefundRechargeOrderInput{
		OrderNo:                entity.OrderNo,
		ExternalOrderID:        firstNonBlank(input.ExternalOrderID, derefString(entity.ExternalOrderID)),
		Channel:                firstNonBlank(input.Channel, entity.Channel),
		CallbackIdempotencyKey: input.CallbackIdempotencyKey,
		CallbackRaw:            input.CallbackRaw,
		Notes:                  derefString(entity.Notes),
		ReversedReason:         "subscription_purchase_refunded",
	})
	if err != nil {
		return nil, nil, err
	}
	if !transitionedToRefunded {
		_ = tx.Rollback()
		return rechargeOrderEntityToService(orderEntity), nil, nil
	}

	if err := s.subscriptionService.ReversePurchasedSubscription(txCtx, entity.UserID, meta.GroupID, meta.ValidityDays, "subscription purchase refunded: "+entity.OrderNo); err != nil {
		return nil, nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}

	return rechargeOrderEntityToService(orderEntity), nil, nil
}

func generateRechargeOrderNo() (string, error) {
	randomPart, err := randomHexString(6)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("RC%s%s", time.Now().UTC().Format("20060102150405"), strings.ToUpper(randomPart)), nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func firstNonBlankPtr(primary *string, fallback *string) *string {
	if primary != nil && strings.TrimSpace(*primary) != "" {
		value := strings.TrimSpace(*primary)
		return &value
	}
	if fallback != nil && strings.TrimSpace(*fallback) != "" {
		value := strings.TrimSpace(*fallback)
		return &value
	}
	return nil
}
