//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type rollingWindowRepoStub struct {
	userSubRepoNoop

	activatedAt    *time.Time
	resetDailyAt   *time.Time
	resetWeeklyAt  *time.Time
	resetMonthlyAt *time.Time
}

func (r *rollingWindowRepoStub) ActivateWindows(_ context.Context, _ int64, start time.Time) error {
	r.activatedAt = &start
	return nil
}

func (r *rollingWindowRepoStub) ResetDailyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetDailyAt = &start
	return nil
}

func (r *rollingWindowRepoStub) ResetWeeklyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetWeeklyAt = &start
	return nil
}

func (r *rollingWindowRepoStub) ResetMonthlyUsage(_ context.Context, _ int64, start time.Time) error {
	r.resetMonthlyAt = &start
	return nil
}

func TestAdvanceRollingWindowStart_AlignsToElapsedWindows(t *testing.T) {
	start := time.Date(2026, 4, 1, 15, 30, 0, 0, time.UTC)
	now := start.Add(49 * time.Hour)

	got := advanceRollingWindowStart(&start, subscriptionDailyWindowDuration, now)

	require.Equal(t, start.Add(48*time.Hour), got)
}

func TestAdvanceRollingWindowStart_UsesCurrentTimeWhenWindowMissing(t *testing.T) {
	now := time.Date(2026, 4, 7, 9, 45, 0, 0, time.UTC)

	got := advanceRollingWindowStart(nil, subscriptionDailyWindowDuration, now)

	require.Equal(t, now, got)
}

func TestCheckAndActivateWindow_UsesCurrentTimeAsRollingAnchor(t *testing.T) {
	now := time.Date(2026, 4, 7, 15, 23, 45, 0, time.UTC)
	repo := &rollingWindowRepoStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	svc.now = func() time.Time { return now }

	err := svc.CheckAndActivateWindow(context.Background(), &UserSubscription{ID: 1})

	require.NoError(t, err)
	require.NotNil(t, repo.activatedAt)
	require.Equal(t, now, *repo.activatedAt)
}

func TestCheckAndResetWindows_AdvancesEachWindowWithoutDrift(t *testing.T) {
	now := time.Date(2026, 4, 7, 18, 0, 0, 0, time.UTC)
	dailyStart := time.Date(2026, 4, 5, 15, 0, 0, 0, time.UTC)
	weeklyStart := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	monthlyStart := time.Date(2026, 2, 5, 9, 30, 0, 0, time.UTC)
	repo := &rollingWindowRepoStub{}
	svc := NewSubscriptionService(groupRepoNoop{}, repo, nil, nil, nil)
	svc.now = func() time.Time { return now }
	sub := &UserSubscription{
		ID:                 1,
		UserID:             10,
		GroupID:            20,
		DailyWindowStart:   &dailyStart,
		WeeklyWindowStart:  &weeklyStart,
		MonthlyWindowStart: &monthlyStart,
		DailyUsageUSD:      10,
		WeeklyUsageUSD:     20,
		MonthlyUsageUSD:    30,
	}

	err := svc.CheckAndResetWindows(context.Background(), sub)

	require.NoError(t, err)
	require.NotNil(t, repo.resetDailyAt)
	require.NotNil(t, repo.resetWeeklyAt)
	require.NotNil(t, repo.resetMonthlyAt)
	require.Equal(t, time.Date(2026, 4, 7, 15, 0, 0, 0, time.UTC), *repo.resetDailyAt)
	require.Equal(t, time.Date(2026, 4, 4, 12, 0, 0, 0, time.UTC), *repo.resetWeeklyAt)
	require.Equal(t, time.Date(2026, 3, 7, 9, 30, 0, 0, time.UTC), *repo.resetMonthlyAt)
	require.Equal(t, 0.0, sub.DailyUsageUSD)
	require.Equal(t, 0.0, sub.WeeklyUsageUSD)
	require.Equal(t, 0.0, sub.MonthlyUsageUSD)
	require.Equal(t, *repo.resetDailyAt, *sub.DailyWindowStart)
	require.Equal(t, *repo.resetWeeklyAt, *sub.WeeklyWindowStart)
	require.Equal(t, *repo.resetMonthlyAt, *sub.MonthlyWindowStart)
}
