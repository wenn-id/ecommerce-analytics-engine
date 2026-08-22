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

// metaAPIConnector fetches real data from the Meta Marketing API (Graph API).
//
// Required environment variables:
//   - META_ACCESS_TOKEN  – system-user access token with ads_read
//   - META_AD_ACCOUNT_ID – ad account id, with or without the "act_" prefix
//
// Optional:
//   - META_API_BASE   (default https://graph.facebook.com)
//   - META_API_VERSION (default v19.0)
//
// Meta exposes ad performance only; FetchDailySales returns no rows because
// shop-level sales are not part of the Marketing API.
type metaAPIConnector struct {
	client      *apiClient
	apiVersion  string
	accessToken string
	accountID   string // normalized to the bare numeric id
}

// NewMetaAPIConnector returns a real Meta connector. It returns an error when
// the required credentials are missing so the factory can fall back to the
// mock connector with a clear log line.
func NewMetaAPIConnector(accessToken, adAccountID, apiBase, apiVersion string) (PlatformConnector, error) {
	if accessToken == "" || adAccountID == "" {
		return nil, fmt.Errorf("META_ACCESS_TOKEN and META_AD_ACCOUNT_ID are required for the real Meta connector")
	}
	if apiBase == "" {
		apiBase = "https://graph.facebook.com"
	}
	if apiVersion == "" {
		apiVersion = "v19.0"
	}
	return &metaAPIConnector{
		client:      newAPIClient(strings.TrimSuffix(apiBase, "/")),
		apiVersion:  apiVersion,
		accessToken: accessToken,
		accountID:   strings.TrimPrefix(adAccountID, "act_"),
	}, nil
}

func (m *metaAPIConnector) GetChannelCode() string { return "meta_ads" }
func (m *metaAPIConnector) GetChannelName() string { return "Meta Ads" }

// authHeader sends the access token as a bearer credential. The Graph API
// accepts it in the Authorization header, which keeps the secret out of the
// request URL and therefore out of error messages and logs
// (query-parameter auth leaks via URL-based diagnostics).
func (m *metaAPIConnector) authHeader() http.Header {
	h := http.Header{}
	h.Set("Authorization", "Bearer "+m.accessToken)
	return h
}

// nextPage rewrites a Graph API paging.next URL into the path/query for the
// next request against the configured API host, dropping any credential that
// Meta embedded in the link.
func (m *metaAPIConnector) nextPage(raw string) (string, url.Values, bool) {
	next, err := url.Parse(raw)
	if err != nil || next.Path == "" {
		return "", nil, false
	}
	q := next.Query()
	q.Del("access_token")
	return next.Path, q, true
}

type metaPaging struct {
	Next string `json:"next"`
}

type metaCampaignsResponse struct {
	Data   []metaCampaign `json:"data"`
	Paging metaPaging     `json:"paging"`
}

type metaCampaign struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	EffectiveStatus string `json:"effective_status"`
	DailyBudget     string `json:"daily_budget"` // minor units, as string
}

func (m *metaAPIConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	var collected []model.Campaign
	query := url.Values{
		"fields": {"id,name,effective_status,daily_budget"},
		"limit":  {"100"},
	}
	path := fmt.Sprintf("/%s/act_%s/campaigns", m.apiVersion, m.accountID)

	for {
		var resp metaCampaignsResponse
		if err := m.client.getJSON(ctx, path, query, m.authHeader(), &resp); err != nil {
			return nil, err
		}
		for _, c := range resp.Data {
			status := c.EffectiveStatus
			if status == "" {
				status = c.Status
			}
			collected = append(collected, model.Campaign{
				ExternalID:  c.ID,
				Name:        c.Name,
				Status:      strings.ToUpper(status),
				DailyBudget: metaMinorUnitsToFloat(c.DailyBudget),
				CreatedAt:   time.Now(),
			})
		}
		if resp.Paging.Next == "" {
			break
		}
		nextPath, nextQuery, ok := m.nextPage(resp.Paging.Next)
		if !ok {
			break
		}
		path, query = nextPath, nextQuery
	}
	if collected == nil {
		collected = []model.Campaign{}
	}
	return collected, nil
}

type metaInsightsResponse struct {
	Data   []metaInsight `json:"data"`
	Paging metaPaging    `json:"paging"`
}

type metaInsight struct {
	CampaignID   string          `json:"campaign_id"`
	DateStart    string          `json:"date_start"`
	DateStop     string          `json:"date_stop"`
	Impressions  json.Number     `json:"impressions"`
	Clicks       json.Number     `json:"clicks"`
	Spend        string          `json:"spend"`
	Actions      []metaAction    `json:"actions"`
	ActionValues []metaActionVal `json:"action_values"`
}

type metaAction struct {
	ActionType string `json:"action_type"`
	Value      string `json:"value"`
}

type metaActionVal struct {
	ActionType string      `json:"action_type"`
	Value      json.Number `json:"value"`
}

// FetchDailyAdMetrics pulls campaign-level insights with time_increment=1.
// The Graph API caps time_increment=1 ranges at 37 days, so longer windows
// are chunked (#62).
func (m *metaAPIConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	var all []AdMetric
	windowStart := startDate
	for windowStart.Before(endDate) || windowStart.Equal(endDate) {
		windowEnd := windowStart.AddDate(0, 0, 36)
		if windowEnd.After(endDate) {
			windowEnd = endDate
		}

		timeRange := fmt.Sprintf(`{"since":"%s","until":"%s"}`,
			windowStart.Format("2006-01-02"), windowEnd.Format("2006-01-02"))
		query := url.Values{
			"level":          {"campaign"},
			"fields":         {"campaign_id,impressions,clicks,spend,actions,action_values"},
			"time_increment": {"1"},
			"time_range":     {timeRange},
			"limit":          {"500"},
		}
		path := fmt.Sprintf("/%s/act_%s/insights", m.apiVersion, m.accountID)

		for {
			var resp metaInsightsResponse
			if err := m.client.getJSON(ctx, path, query, m.authHeader(), &resp); err != nil {
				return nil, err
			}
			for _, row := range resp.Data {
				metric, err := row.toAdMetric()
				if err != nil {
					continue
				}
				all = append(all, metric)
			}
			if resp.Paging.Next == "" {
				break
			}
			nextPath, nextQuery, ok := m.nextPage(resp.Paging.Next)
			if !ok {
				break
			}
			path, query = nextPath, nextQuery
		}

		windowStart = windowEnd.AddDate(0, 0, 1)
	}
	return all, nil
}

func (row metaInsight) toAdMetric() (AdMetric, error) {
	date, err := time.Parse("2006-01-02", row.DateStart)
	if err != nil {
		return AdMetric{}, err
	}
	impressions, _ := row.Impressions.Int64()
	clicks, _ := row.Clicks.Int64()
	spend, _ := strconv.ParseFloat(row.Spend, 64)

	var conversions int64
	for _, a := range row.Actions {
		if a.ActionType == "purchase" || a.ActionType == "offsite_conversion.fb_pixel_purchase" || a.ActionType == "omni_purchase" {
			if v, err := strconv.ParseInt(a.Value, 10, 64); err == nil {
				conversions += v
			}
		}
	}
	var attributedRevenue float64
	for _, av := range row.ActionValues {
		if av.ActionType == "purchase" || av.ActionType == "omni_purchase" {
			if v, err := av.Value.Float64(); err == nil {
				attributedRevenue += v
			}
		}
	}

	return AdMetric{
		CampaignExternalID: row.CampaignID,
		Date:               date,
		Impressions:        impressions,
		Clicks:             clicks,
		Spend:              spend,
		Conversions:        conversions,
		AttributedRevenue:  attributedRevenue,
	}, nil
}

func (m *metaAPIConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	// Marketing API has no shop sales endpoint; sales for Meta traffic are
	// tracked by the shop platform itself.
	return []model.DailySalesMetric{}, nil
}

func metaMinorUnitsToFloat(minor string) float64 {
	if minor == "" {
		return 0
	}
	v, err := strconv.ParseFloat(minor, 64)
	if err != nil {
		return 0
	}
	return v / 100
}
