package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
)

func TestOverviewMetricsJSON(t *testing.T) {
	metrics := model.OverviewMetrics{
		TotalSpend:  15000000.0,
		TotalGMV:    75000000.0,
		BlendedROAS: 5.0,
		TotalOrders: 350,
		AvgCPA:      42857.14,
		ACOS:        20.0,
		NetMargin:   32500000.0,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("failed to marshal metrics: %v", err)
	}

	var parsed model.OverviewMetrics
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal metrics: %v", err)
	}

	if parsed.BlendedROAS != 5.0 || parsed.TotalOrders != 350 {
		t.Errorf("unexpected unmarshaled values: %+v", parsed)
	}
}

func TestDailyAdMetricValidation(t *testing.T) {
	validMetric := model.DailyAdMetric{
		CampaignID:        1,
		Date:              time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		Impressions:       10000,
		Clicks:            500,
		Spend:             250000.0,
		Conversions:       15,
		AttributedRevenue: 1500000.0,
	}

	if err := validMetric.Validate(); err != nil {
		t.Errorf("expected valid metric to pass validation, got: %v", err)
	}

	invalidSpend := validMetric
	invalidSpend.Spend = -100.0
	if err := invalidSpend.Validate(); err != model.ErrNegativeSpend {
		t.Errorf("expected ErrNegativeSpend, got: %v", err)
	}

	invalidClicks := validMetric
	invalidClicks.Clicks = 20000
	if err := invalidClicks.Validate(); err != model.ErrClicksExceedImps {
		t.Errorf("expected ErrClicksExceedImps, got: %v", err)
	}

	invalidCampaign := validMetric
	invalidCampaign.CampaignID = 0
	if err := invalidCampaign.Validate(); err != model.ErrInvalidCampaignID {
		t.Errorf("expected ErrInvalidCampaignID, got: %v", err)
	}
}

func TestDailySalesMetricValidation(t *testing.T) {
	validSales := model.DailySalesMetric{
		ChannelID:      1,
		Date:           time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		TotalOrders:    50,
		GMV:            10000000,
		NetSales:       9500000,
		COGS:           4500000,
		ReturnedOrders: 2,
	}

	if err := validSales.Validate(); err != nil {
		t.Errorf("expected valid sales metric to pass validation, got: %v", err)
	}

	invalidOrders := validSales
	invalidOrders.TotalOrders = -5
	if err := invalidOrders.Validate(); err != model.ErrNegativeOrders {
		t.Errorf("expected ErrNegativeOrders, got: %v", err)
	}
}
