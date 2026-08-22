package connector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ecommerce-analytics/internal/model"
)

// tikTokAPIConnector fetches real data from the TikTok Business (Ads) API v1.3.
//
// Required environment variables:
//   - TIKTOK_ACCESS_TOKEN  – advertiser-approved access token
//   - TIKTOK_ADVERTISER_ID – numeric advertiser id
//
// Optional:
//   - TIKTOK_API_BASE (default https://business-api.tiktok.com)
//
// The Ads API covers campaign/ad performance only. TikTok *Shop* sales live
// on a separate signed Shop API surface; FetchDailySales returns no rows
// until that integration is added.
type tikTokAPIConnector struct {
	client       *apiClient
	accessToken  string
	advertiserID string
}

func NewTikTokAPIConnector(accessToken, advertiserID, apiBase string) (PlatformConnector, error) {
	if accessToken == "" || advertiserID == "" {
		return nil, fmt.Errorf("TIKTOK_ACCESS_TOKEN and TIKTOK_ADVERTISER_ID are required for the real TikTok connector")
	}
	if apiBase == "" {
		apiBase = "https://business-api.tiktok.com"
	}
	return &tikTokAPIConnector{
		client:       newAPIClient(strings.TrimSuffix(apiBase, "/")),
		accessToken:  accessToken,
		advertiserID: advertiserID,
	}, nil
}

func (t *tikTokAPIConnector) GetChannelCode() string { return "tiktok_shop" }
func (t *tikTokAPIConnector) GetChannelName() string { return "TikTok Shop" }

func (t *tikTokAPIConnector) authHeader() http.Header {
	h := http.Header{}
	h.Set("Access-Token", t.accessToken)
	return h
}

type tikTokResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func (t *tikTokResponse) check() error {
	if t.Code != 0 {
		return fmt.Errorf("TikTok API error %d: %s", t.Code, t.Message)
	}
	return nil
}

type tikTokCampaignPage struct {
	List     []tikTokCampaign `json:"list"`
	PageInfo struct {
		TotalPage int `json:"total_page"`
		PageSize  int `json:"page_size"`
	} `json:"page_info"`
}

type tikTokCampaign struct {
	CampaignID   string `json:"campaign_id"`
	CampaignName string `json:"campaign_name"`
	Status       string `json:"status"` // e.g. ENABLE / DISABLE / DELETE
	// Budget fields are minor units of the account currency.
	LifetimeBudget json.Number `json:"lifetime_budget"`
	DailyBudget    json.Number `json:"daily_budget"`
	CreateTime     json.Number `json:"create_time"` // unix seconds
}

func (t *tikTokAPIConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	var collected []model.Campaign
	const pageSize = 100
	for page := 1; ; page++ {
		query := url.Values{
			"advertiser_id": {t.advertiserID},
			"page":          {strconv.Itoa(page)},
			"page_size":     {strconv.Itoa(pageSize)},
		}
		var resp tikTokResponse
		if err := t.client.getJSON(ctx, "/open_api/v1.3/campaign/get/", query, t.authHeader(), &resp); err != nil {
			return nil, err
		}
		if err := resp.check(); err != nil {
			return nil, err
		}
		var pageData tikTokCampaignPage
		if len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, &pageData); err != nil {
				return nil, fmt.Errorf("decode TikTok campaigns: %w", err)
			}
		}
		for _, c := range pageData.List {
			createdAt := time.Now()
			if secs, err := c.CreateTime.Int64(); err == nil && secs > 0 {
				createdAt = time.Unix(secs, 0)
			}
			collected = append(collected, model.Campaign{
				ExternalID:  c.CampaignID,
				Name:        c.CampaignName,
				Status:      normalizeTikTokStatus(c.Status),
				DailyBudget: jsonNumberToFloat(c.DailyBudget) / 100,
				CreatedAt:   createdAt,
			})
		}
		if page >= pageData.PageInfo.TotalPage || len(pageData.List) == 0 {
			break
		}
	}
	if collected == nil {
		collected = []model.Campaign{}
	}
	return collected, nil
}

type tikTokReportPage struct {
	List     []map[string]string `json:"list"`
	PageInfo struct {
		TotalPage int `json:"total_page"`
	} `json:"page_info"`
}

// FetchDailyAdMetrics queries the integrated reporting endpoint at
// campaign/day granularity. Metric values arrive as strings and are parsed
// defensively (#62).
func (t *tikTokAPIConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	dimensions := `["campaign_id","stat_time_day"]`
	metrics := `["impressions","clicks","spend","conversions","total_complete_payment_value"]`

	var all []AdMetric
	const pageSize = 200
	for page := 1; ; page++ {
		query := url.Values{
			"report_type":   {"BASIC"},
			"data_level":    {"AUCTION_CAMPAIGN"},
			"dimensions":    {dimensions},
			"metrics":       {metrics},
			"advertiser_id": {t.advertiserID},
			"start_date":    {startDate.Format("2006-01-02")},
			"end_date":      {endDate.Format("2006-01-02")},
			"page":          {strconv.Itoa(page)},
			"page_size":     {strconv.Itoa(pageSize)},
		}
		var resp tikTokResponse
		if err := t.client.getJSON(ctx, "/open_api/v1.3/report/integrated/get/", query, t.authHeader(), &resp); err != nil {
			return nil, err
		}
		if err := resp.check(); err != nil {
			return nil, err
		}
		var pageData tikTokReportPage
		if len(resp.Data) > 0 {
			if err := json.Unmarshal(resp.Data, &pageData); err != nil {
				return nil, fmt.Errorf("decode TikTok report: %w", err)
			}
		}
		for _, row := range pageData.List {
			metric, err := tikTokRowToAdMetric(row)
			if err != nil {
				continue
			}
			all = append(all, metric)
		}
		if page >= pageData.PageInfo.TotalPage || len(pageData.List) == 0 {
			break
		}
	}
	return all, nil
}

func tikTokRowToAdMetric(row map[string]string) (AdMetric, error) {
	// stat_time_day looks like "2024-05-01 00:00:00" in the reporting tz.
	raw := row["stat_time_day"]
	if len(raw) >= 10 {
		raw = raw[:10]
	}
	date, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return AdMetric{}, err
	}
	return AdMetric{
		CampaignExternalID: row["campaign_id"],
		Date:               date,
		Impressions:        parseIntString(row["impressions"]),
		Clicks:             parseIntString(row["clicks"]),
		Spend:              parseFloatString(row["spend"]),
		Conversions:        parseIntString(row["conversions"]),
		AttributedRevenue:  parseFloatString(row["total_complete_payment_value"]),
	}, nil
}

func (t *tikTokAPIConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	// TikTok Shop sales require the separate Shop API credentials and signing
	// scheme; not part of the Ads API token used here.
	return []model.DailySalesMetric{}, nil
}

func normalizeTikTokStatus(status string) string {
	switch strings.ToUpper(status) {
	case "ENABLE":
		return "ACTIVE"
	case "DISABLE":
		return "PAUSED"
	case "DELETE", "DELETED":
		return "DELETED"
	default:
		return strings.ToUpper(status)
	}
}

func parseIntString(s string) int64 {
	v, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return v
}

func parseFloatString(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func jsonNumberToFloat(n json.Number) float64 {
	v, _ := n.Float64()
	return v
}
