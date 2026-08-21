package service

import (
	"context"
	"math"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

type AnalyticsService interface {
	GetOverviewMetrics(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error)
	GetTrendData(ctx context.Context, start, end time.Time) ([]model.TrendDataPoint, error)
	GetChannelBreakdown(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error)
}

type analyticsService struct {
	repo store.Repository
}

func NewAnalyticsService(repo store.Repository) AnalyticsService {
	return &analyticsService{repo: repo}
}

func (s *analyticsService) GetOverviewMetrics(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error) {
	totalSpend, totalGMV, totalCOGS, totalOrders, err := s.repo.GetAggregatedOverview(ctx, start, end)
	if err != nil {
		return nil, err
	}

	var blendedROAS, avgCPA, acos float64
	if totalSpend > 0 {
		blendedROAS = totalGMV / totalSpend
		acos = (totalSpend / math.Max(totalGMV, 1.0)) * 100.0
	}
	if totalOrders > 0 {
		avgCPA = totalSpend / float64(totalOrders)
	}

	platformFees := totalGMV * 0.10
	netMargin := totalGMV - totalSpend - totalCOGS - platformFees

	return &model.OverviewMetrics{
		TotalSpend:  round2(totalSpend),
		TotalGMV:    round2(totalGMV),
		BlendedROAS: round2(blendedROAS),
		TotalOrders: totalOrders,
		AvgCPA:      round2(avgCPA),
		ACOS:        round2(acos),
		NetMargin:   round2(netMargin),
	}, nil
}

func (s *analyticsService) GetTrendData(ctx context.Context, start, end time.Time) ([]model.TrendDataPoint, error) {
	return s.repo.GetAggregatedDailyTrends(ctx, start, end)
}

func (s *analyticsService) GetChannelBreakdown(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error) {
	return s.repo.GetAggregatedChannelSummaries(ctx, start, end)
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
