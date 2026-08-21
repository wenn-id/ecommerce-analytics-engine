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
	metric := model.DailyAdMetric{
		CampaignID:        1,
		Date:              time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
		Impressions:       10000,
		Clicks:            500,
		Spend:             250000.0,
		Conversions:       15,
		AttributedRevenue: 1500000.0,
	}

	if metric.Spend < 0 {
		t.Errorf("spend should not be negative")
	}
}
