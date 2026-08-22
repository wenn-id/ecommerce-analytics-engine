package connector_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
)

func TestFactoryFallsBackToMocksWithoutCredentials(t *testing.T) {
	connectors := connector.BuildConnectors(func(string) string { return "" })
	if len(connectors) != 3 {
		t.Fatalf("expected 3 connectors, got %d", len(connectors))
	}
	for _, c := range connectors {
		campaigns, err := c.FetchCampaigns(context.Background())
		if err != nil || len(campaigns) == 0 {
			t.Errorf("%s: expected mock campaigns, got err=%v", c.GetChannelCode(), err)
		}
	}
}

func TestFactoryPrefersRealConnectorWithCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": []interface{}{}, "paging": map[string]string{}})
	}))
	defer srv.Close()

	env := func(key string) string {
		switch key {
		case "META_ACCESS_TOKEN":
			return "token"
		case "META_AD_ACCOUNT_ID":
			return "act_123"
		case "META_API_BASE":
			return srv.URL
		default:
			return ""
		}
	}
	connectors := connector.BuildConnectors(env)
	// TikTok/Shopee still fall back to mocks; Meta hits the test server.
	if _, err := connectors[0].FetchCampaigns(context.Background()); err != nil {
		t.Fatalf("real Meta connector should query the test server, got %v", err)
	}
}

func TestMetaConnectorParsesCampaignsAndInsights(t *testing.T) {
	var insightsCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case endsWith(r.URL.Path, "/campaigns"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]string{
					{"id": "12033060016", "name": "Catalog Sales", "effective_status": "ACTIVE", "daily_budget": "7500000"},
				},
				"paging": map[string]string{},
			})
		case endsWith(r.URL.Path, "/insights"):
			insightsCalls++
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"campaign_id": "12033060016",
						"date_start":  "2026-07-01",
						"impressions": "18000",
						"clicks":      "412",
						"spend":       "1312.45",
						"actions":     []map[string]string{{"action_type": "purchase", "value": "27"}},
						"action_values": []map[string]interface{}{
							{"action_type": "purchase", "value": json.Number("5523.10")},
						},
					},
				},
				"paging": map[string]string{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	conn, err := connector.NewMetaAPIConnector("token", "123", srv.URL, "v19.0")
	if err != nil {
		t.Fatalf("connector: %v", err)
	}

	campaigns, err := conn.FetchCampaigns(context.Background())
	if err != nil {
		t.Fatalf("campaigns: %v", err)
	}
	if len(campaigns) != 1 || campaigns[0].ExternalID != "12033060016" {
		t.Fatalf("unexpected campaigns: %+v", campaigns)
	}
	if campaigns[0].DailyBudget != 75000.0 { // minor units converted
		t.Errorf("expected daily budget 75000, got %v", campaigns[0].DailyBudget)
	}

	// A 40-day window must be chunked into two insights requests.
	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 39)
	metrics, err := conn.FetchDailyAdMetrics(context.Background(), start, end)
	if err != nil {
		t.Fatalf("insights: %v", err)
	}
	if insightsCalls != 2 {
		t.Errorf("expected 2 chunked insight requests for a 40-day window, got %d", insightsCalls)
	}
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metric rows across chunks, got %d", len(metrics))
	}
	m := metrics[0]
	if m.CampaignExternalID != "12033060016" || m.Clicks != 412 || m.Conversions != 27 {
		t.Errorf("unexpected metric: %+v", m)
	}
	if m.Spend != 1312.45 || m.AttributedRevenue != 5523.10 {
		t.Errorf("unexpected money fields: %+v", m)
	}
}

func TestTikTokConnectorParsesReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case endsWith(r.URL.Path, "/campaign/get/"):
			if r.Header.Get("Access-Token") != "tt-token" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0, "message": "OK",
				"data": map[string]interface{}{
					"list": []map[string]interface{}{
						{"campaign_id": "1789567", "campaign_name": "GMV Max", "status": "ENABLE",
							"daily_budget": json.Number("100000"), "create_time": json.Number("1750000000")},
					},
					"page_info": map[string]int{"total_page": 1},
				},
			})
		case endsWith(r.URL.Path, "/report/integrated/get/"):
			if got := r.URL.Query().Get("dimensions"); got == "" {
				http.Error(w, "missing dimensions", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"code": 0, "message": "OK",
				"data": map[string]interface{}{
					"list": []map[string]string{
						{"campaign_id": "1789567", "stat_time_day": "2026-07-01 00:00:00",
							"impressions": "25000", "clicks": "650", "spend": "900.12",
							"conversions": "35", "total_complete_payment_value": "4410.59"},
					},
					"page_info": map[string]int{"total_page": 1},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	conn, err := connector.NewTikTokAPIConnector("tt-token", "73199", srv.URL)
	if err != nil {
		t.Fatalf("connector: %v", err)
	}

	campaigns, err := conn.FetchCampaigns(context.Background())
	if err != nil {
		t.Fatalf("campaigns: %v", err)
	}
	if len(campaigns) != 1 || campaigns[0].Status != "ACTIVE" {
		t.Fatalf("unexpected campaigns: %+v", campaigns)
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	metrics, err := conn.FetchDailyAdMetrics(context.Background(), start, start.AddDate(0, 0, 2))
	if err != nil {
		t.Fatalf("report: %v", err)
	}
	if len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d", len(metrics))
	}
	m := metrics[0]
	if m.CampaignExternalID != "1789567" || m.Clicks != 650 || m.Spend != 900.12 ||
		m.Conversions != 35 || m.AttributedRevenue != 4410.59 {
		t.Errorf("unexpected metric: %+v", m)
	}
}

func TestShopeeConnectorSignsRequestsAndAggregatesOrders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Recompute the expected signature from the query the connector sent.
		partnerID := r.URL.Query().Get("partner_id")
		ts := r.URL.Query().Get("timestamp")
		token := r.URL.Query().Get("access_token")
		shopID := r.URL.Query().Get("shop_id")
		base := partnerID + r.URL.Path + ts + token + shopID
		mac := hmac.New(sha256.New, []byte("partner-key"))
		mac.Write([]byte(base))
		if r.URL.Query().Get("sign") != hex.EncodeToString(mac.Sum(nil)) {
			http.Error(w, "bad sign", http.StatusForbidden)
			return
		}

		switch {
		case endsWith(r.URL.Path, "/ads/get_product_campaign_daily_performance"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "error_success",
				"response": []map[string]interface{}{
					{"campaign_list": []map[string]interface{}{
						{"campaign_id": json.Number("9001"), "metrics_list": []map[string]interface{}{
							{"date": "2026-07-01", "impression": json.Number("12000"), "clicks": json.Number("310"),
								"expense": 512.4, "direct_gmv": 2100.0, "broad_gmv": 321.5},
						}},
					}},
				},
			})
		case endsWith(r.URL.Path, "/order/get_order_list"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "error_success",
				"response": map[string]interface{}{
					"more": false,
					"order_list": []map[string]string{
						{"order_sn": "SN1", "order_status": "COMPLETED"},
						{"order_sn": "SN2", "order_status": "CANCELLED"},
					},
				},
			})
		case endsWith(r.URL.Path, "/order/get_order_detail"):
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": "error_success",
				"response": map[string]interface{}{
					"order_list": []map[string]interface{}{
						{"order_sn": "SN1", "total_amount": 150000.0, "create_time": json.Number(strconv.FormatInt(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC).Unix(), 10))},
						{"order_sn": "SN2", "total_amount": 99000.0, "create_time": json.Number(strconv.FormatInt(time.Date(2026, 7, 1, 11, 0, 0, 0, time.UTC).Unix(), 10))},
					},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	conn, err := connector.NewShopeeAPIConnector("12345", "partner-key", "shop-token", "999", srv.URL, "9001")
	if err != nil {
		t.Fatalf("connector: %v", err)
	}

	start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)

	ads, err := conn.FetchDailyAdMetrics(context.Background(), start, end)
	if err != nil {
		t.Fatalf("ads: %v", err)
	}
	if len(ads) != 1 {
		t.Fatalf("expected 1 ad metric, got %d", len(ads))
	}
	if ads[0].CampaignExternalID != "9001" || ads[0].Spend != 512.4 || ads[0].AttributedRevenue != 2421.5 {
		t.Errorf("unexpected ad metric: %+v", ads[0])
	}

	sales, err := conn.FetchDailySales(context.Background(), start, end)
	if err != nil {
		t.Fatalf("sales: %v", err)
	}
	if len(sales) != 2 {
		t.Fatalf("expected 2 daily sales rows (window fill), got %d", len(sales))
	}
	day1 := sales[0]
	if day1.TotalOrders != 1 || day1.GMV != 150000.0 || day1.ReturnedOrders != 1 {
		t.Errorf("unexpected day-1 sales: %+v", day1)
	}
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
