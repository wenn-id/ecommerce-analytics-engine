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
	adMetrics, err := s.repo.QueryAdMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}

	salesMetrics, err := s.repo.QuerySalesMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}

	var totalSpend float64
	for _, m := range adMetrics {
		totalSpend += m.Spend
	}

	var totalGMV, totalCOGS float64
	var totalOrders int
	for _, sm := range salesMetrics {
		totalGMV += sm.GMV
		totalCOGS += sm.COGS
		totalOrders += sm.TotalOrders
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
	adMetrics, err := s.repo.QueryAdMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}
	salesMetrics, err := s.repo.QuerySalesMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}

	dailyMap := make(map[string]*model.TrendDataPoint)
	curr := start
	for !curr.After(end) {
		dStr := curr.Format("2006-01-02")
		dailyMap[dStr] = &model.TrendDataPoint{Date: dStr}
		curr = curr.AddDate(0, 0, 1)
	}

	for _, a := range adMetrics {
		dStr := a.Date.Format("2006-01-02")
		if pt, ok := dailyMap[dStr]; ok {
			pt.Spend += a.Spend
		}
	}

	for _, sm := range salesMetrics {
		dStr := sm.Date.Format("2006-01-02")
		if pt, ok := dailyMap[dStr]; ok {
			pt.GMV += sm.GMV
			pt.TotalOrders += sm.TotalOrders
		}
	}

	var results []model.TrendDataPoint
	curr = start
	for !curr.After(end) {
		dStr := curr.Format("2006-01-02")
		pt := dailyMap[dStr]
		if pt.Spend > 0 {
			pt.BlendedROAS = round2(pt.GMV / pt.Spend)
		}
		pt.Spend = round2(pt.Spend)
		pt.GMV = round2(pt.GMV)
		results = append(results, *pt)
		curr = curr.AddDate(0, 0, 1)
	}

	return results, nil
}

func (s *analyticsService) GetChannelBreakdown(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error) {
	channels, err := s.repo.GetChannels(ctx)
	if err != nil {
		return nil, err
	}

	adMetrics, err := s.repo.QueryAdMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}
	salesMetrics, err := s.repo.QuerySalesMetrics(ctx, start, end)
	if err != nil {
		return nil, err
	}

	var totalSpend, totalGMV float64
	for _, a := range adMetrics {
		totalSpend += a.Spend
	}
	for _, sm := range salesMetrics {
		totalGMV += sm.GMV
	}

	var summaries []model.ChannelSummary
	for _, ch := range channels {
		var chSpend, chGMV float64
		for _, sm := range salesMetrics {
			if sm.ChannelID == ch.ID {
				chGMV += sm.GMV
			}
		}
		for _, a := range adMetrics {
			if a.CampaignID == ch.ID {
				chSpend += a.Spend
			}
		}

		var roas float64
		if chSpend > 0 {
			roas = chGMV / chSpend
		}

		summaries = append(summaries, model.ChannelSummary{
			ChannelCode:       ch.Code,
			ChannelName:       ch.Name,
			TotalSpend:        round2(chSpend),
			TotalGMV:          round2(chGMV),
			ChannelROAS:       round2(roas),
			SpendSharePercent: round2((chSpend / math.Max(totalSpend, 1.0)) * 100.0),
			GMVSharePercent:   round2((chGMV / math.Max(totalGMV, 1.0)) * 100.0),
		})
	}
	return summaries, nil
}

func round2(val float64) float64 {
	return math.Round(val*100) / 100
}
