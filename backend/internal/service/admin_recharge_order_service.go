package service

import (
	"context"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/rechargeorder"
	"github.com/Wei-Shaw/sub2api/ent/referralcommission"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
)

type AdminListRechargeOrdersInput struct {
	Page           int
	PageSize       int
	Status         string
	Channel        string
	Search         string
	StartDate      *time.Time
	EndDate        *time.Time
	WithCommission bool
	RefundedOnly   bool
}

func (s *ReferralService) ListAdminRechargeOrders(ctx context.Context, input *AdminListRechargeOrdersInput) ([]AdminRechargeOrderDetail, int64, error) {
	if s.entClient == nil {
		return nil, 0, ErrServiceUnavailable
	}
	if input == nil {
		input = &AdminListRechargeOrdersInput{}
	}

	page, pageSize := normalizePage(input.Page, input.PageSize)
	baseQuery := s.buildAdminRechargeOrderQuery(input)

	total, err := baseQuery.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := baseQuery.
		Order(dbent.Desc(rechargeorder.FieldID)).
		WithUser().
		WithReferralCommissions().
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]AdminRechargeOrderDetail, 0, len(items))
	for _, item := range items {
		detail := AdminRechargeOrderDetail{
			RechargeOrder: *rechargeOrderEntityToService(item),
			User:          entUserToService(item.Edges.User),
		}
		for _, commission := range item.Edges.ReferralCommissions {
			detail.CommissionCount++
			detail.TotalCommissionAmount = roundMoney(detail.TotalCommissionAmount + commission.CommissionAmount)
			switch commission.Status {
			case ReferralCommissionStatusRecorded:
				detail.RecordedCommissionAmount = roundMoney(detail.RecordedCommissionAmount + commission.CommissionAmount)
			case ReferralCommissionStatusReversed:
				detail.ReversedCommissionAmount = roundMoney(detail.ReversedCommissionAmount + commission.CommissionAmount)
			}
		}
		result = append(result, detail)
	}

	return result, int64(total), nil
}

func (s *ReferralService) GetAdminRechargeOrderStats(ctx context.Context, input *AdminListRechargeOrdersInput) (*AdminRechargeOrderStats, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if input == nil {
		input = &AdminListRechargeOrdersInput{}
	}

	items, err := s.buildAdminRechargeOrderQuery(input).
		WithReferralCommissions().
		All(ctx)
	if err != nil {
		return nil, err
	}

	stats := &AdminRechargeOrderStats{
		TotalOrders: int64(len(items)),
	}
	for _, item := range items {
		switch item.Status {
		case RechargeOrderStatusPending:
			stats.PendingOrders++
		case RechargeOrderStatusPaid:
			stats.PaidOrders++
			stats.TotalPaidAmount = roundMoney(stats.TotalPaidAmount + item.Amount)
		case RechargeOrderStatusFailed:
			stats.FailedOrders++
		case RechargeOrderStatusRefunded:
			stats.RefundedOrders++
			stats.TotalRefundedAmount = roundMoney(stats.TotalRefundedAmount + item.Amount)
		}
		for _, commission := range item.Edges.ReferralCommissions {
			if commission.Status == ReferralCommissionStatusRecorded {
				stats.TotalCommissionAmount = roundMoney(stats.TotalCommissionAmount + commission.CommissionAmount)
			}
		}
	}

	return stats, nil
}

func (s *ReferralService) GetAdminRechargeOrderDetail(ctx context.Context, orderID int64) (*AdminRechargeOrderDetail, error) {
	if s.entClient == nil {
		return nil, ErrServiceUnavailable
	}
	if orderID <= 0 {
		return nil, ErrRechargeOrderInvalid
	}

	item, err := s.entClient.RechargeOrder.Query().
		Where(rechargeorder.IDEQ(orderID)).
		WithUser().
		WithReferralCommissions(func(query *dbent.ReferralCommissionQuery) {
			query.WithReferredUser()
		}).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, ErrRechargeOrderNotFound
		}
		return nil, err
	}

	detail := &AdminRechargeOrderDetail{
		RechargeOrder: *rechargeOrderEntityToService(item),
		User:          entUserToService(item.Edges.User),
	}
	for _, commission := range item.Edges.ReferralCommissions {
		detail.CommissionCount++
		detail.TotalCommissionAmount = roundMoney(detail.TotalCommissionAmount + commission.CommissionAmount)
		switch commission.Status {
		case ReferralCommissionStatusRecorded:
			detail.RecordedCommissionAmount = roundMoney(detail.RecordedCommissionAmount + commission.CommissionAmount)
		case ReferralCommissionStatusReversed:
			detail.ReversedCommissionAmount = roundMoney(detail.ReversedCommissionAmount + commission.CommissionAmount)
		}
	}

	return detail, nil
}

func (s *ReferralService) buildAdminRechargeOrderQuery(input *AdminListRechargeOrdersInput) *dbent.RechargeOrderQuery {
	query := s.entClient.RechargeOrder.Query()
	if input == nil {
		return query
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	switch status {
	case RechargeOrderStatusPending, RechargeOrderStatusPaid, RechargeOrderStatusFailed, RechargeOrderStatusRefunded:
		query = query.Where(rechargeorder.StatusEQ(status))
	}

	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	if channel != "" {
		query = query.Where(rechargeorder.ChannelEQ(channel))
	}
	if input.RefundedOnly {
		query = query.Where(rechargeorder.StatusEQ(RechargeOrderStatusRefunded))
	}
	if input.WithCommission {
		query = query.Where(rechargeorder.HasReferralCommissionsWith(referralcommission.StatusEQ(ReferralCommissionStatusRecorded)))
	}
	if input.StartDate != nil {
		query = query.Where(rechargeorder.CreatedAtGTE(*input.StartDate))
	}
	if input.EndDate != nil {
		endExclusive := input.EndDate.Add(24 * time.Hour)
		query = query.Where(rechargeorder.CreatedAtLT(endExclusive))
	}

	search := strings.TrimSpace(input.Search)
	if search != "" {
		query = query.Where(
			rechargeorder.Or(
				rechargeorder.OrderNoContainsFold(search),
				rechargeorder.ExternalOrderIDContainsFold(search),
				rechargeorder.HasUserWith(
					dbuser.Or(
						dbuser.EmailContainsFold(search),
						dbuser.UsernameContainsFold(search),
					),
				),
			),
		)
	}

	return query
}
