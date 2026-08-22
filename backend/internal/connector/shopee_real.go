package connector

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ecommerce-analytics/internal/model"
)

// shopeeAPIConnector fetches real data from the Shopee Open Platform v2 API.
//
// Required environment variables:
//   - SHOPEE_PARTNER_ID    – numeric partner id
//   - SHOPEE_PARTNER_KEY   – partner key used for request signing
//   - SHOPEE_ACCESS_TOKEN  – shop-authorized merchant token (obtained via the
//     OAuth authorization flow; tokens cannot be minted server-side alone)
//   - SHOPEE_SHOP_ID       – numeric shop id
//
// Optional:
//   - SHOPEE_API_BASE   (default https://openplatform.shopee.com)
//   - SHOPEE_CAMPAIGN_IDS – comma-separated Shopee Ads campaign ids to sync.
//     The Ads API resolves campaign metadata by id (there is no
//     list-everything endpoint), so operators pin the campaigns they own.
//   - SHOPEE_TIMEZONE  – IANA name of the shop's reporting timezone
//     (default UTC). Used consistently for the query window and for bucketing
//     order timestamps into days, so near-midnight orders land on the same
//     day the seller sees in Shopee.
//
// Ads metrics come from get_product_campaign_daily_performance. Sales are
// aggregated from get_order_list + get_order_detail. Shopee reports order
// totals only; COGS and net sales are not part of the order payload, so COGS
// is recorded as 0 and NetSales mirrors GMV.
type shopeeAPIConnector struct {
	client      *apiClient
	partnerID   string
	partnerKey  string
	accessToken string
	shopID      string
	campaignIDs []string
	loc         *time.Location
}

func NewShopeeAPIConnector(partnerID, partnerKey, accessToken, shopID, apiBase, campaignIDs, timezone string) (PlatformConnector, error) {
	if partnerID == "" || partnerKey == "" || accessToken == "" || shopID == "" {
		return nil, fmt.Errorf("SHOPEE_PARTNER_ID, SHOPEE_PARTNER_KEY, SHOPEE_ACCESS_TOKEN and SHOPEE_SHOP_ID are required for the real Shopee connector")
	}
	if apiBase == "" {
		apiBase = "https://openplatform.shopee.com"
	}
	loc, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil || loc == nil {
		if strings.TrimSpace(timezone) != "" {
			// An explicitly configured but unparseable timezone must not
			// silently shift reporting days; fall back to UTC and say so.
			slog.Warn("invalid SHOPEE_TIMEZONE, falling back to UTC", "value", timezone, "error", err)
		}
		loc = time.UTC
	}
	var campaigns []string
	for _, id := range strings.Split(campaignIDs, ",") {
		if id = strings.TrimSpace(id); id != "" {
			campaigns = append(campaigns, id)
		}
	}
	return &shopeeAPIConnector{
		client:      newAPIClient(strings.TrimSuffix(apiBase, "/")),
		partnerID:   partnerID,
		partnerKey:  partnerKey,
		accessToken: accessToken,
		shopID:      shopID,
		campaignIDs: campaigns,
		loc:         loc,
	}, nil
}

// reportingDay reinterprets a calendar date in the shop's reporting timezone
// (the incoming timestamps carry wall-clock dates from the sync window).
func (s *shopeeAPIConnector) reportingDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, s.loc)
}

func (s *shopeeAPIConnector) GetChannelCode() string { return "shopee" }
func (s *shopeeAPIConnector) GetChannelName() string { return "Shopee" }

// sign builds the Shopee v2 signature: HMAC-SHA256(partnerKey,
// partnerID + path + timestamp + accessToken + shopID).
func (s *shopeeAPIConnector) sign(path string, timestamp int64) string {
	var b strings.Builder
	b.WriteString(s.partnerID)
	b.WriteString(path)
	b.WriteString(strconv.FormatInt(timestamp, 10))
	b.WriteString(s.accessToken)
	b.WriteString(s.shopID)
	mac := hmac.New(sha256.New, []byte(s.partnerKey))
	mac.Write([]byte(b.String()))
	return hex.EncodeToString(mac.Sum(nil))
}

// signedQuery assembles the common+signed query parameters for a shop-level
// GET request.
func (s *shopeeAPIConnector) signedQuery(path string, extra url.Values) url.Values {
	ts := time.Now().Unix()
	q := url.Values{}
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("partner_id", s.partnerID)
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("access_token", s.accessToken)
	q.Set("shop_id", s.shopID)
	q.Set("sign", s.sign(path, ts))
	return q
}

type shopeeBaseResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type shopeeCampaignSettingResponse struct {
	shopeeBaseResponse
	Response struct {
		Campaigns []struct {
			CampaignID   json.Number `json:"campaign_id"`
			CampaignName string      `json:"campaign_name"`
			Status       string      `json:"status"`
			Budget       float64     `json:"budget"`
		} `json:"campaign_list"`
	} `json:"response"`
}

func (s *shopeeAPIConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	if len(s.campaignIDs) == 0 {
		return []model.Campaign{}, nil
	}
	path := "/api/v2/ads/get_product_level_campaign_setting_info"
	query := s.signedQuery(path, url.Values{
		"campaign_id_list": {strings.Join(s.campaignIDs, ",")},
		"info_type_list":   {"campaign_basic_info"},
	})
	var resp shopeeCampaignSettingResponse
	if err := s.client.getJSON(ctx, path, query, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Error != "" && resp.Error != "error_success" {
		return nil, fmt.Errorf("Shopee campaign settings API %s: %s", resp.Error, resp.Message)
	}
	campaigns := make([]model.Campaign, 0, len(resp.Response.Campaigns))
	for _, c := range resp.Response.Campaigns {
		campaigns = append(campaigns, model.Campaign{
			ExternalID:  c.CampaignID.String(),
			Name:        c.CampaignName,
			Status:      strings.ToUpper(c.Status),
			DailyBudget: c.Budget,
			CreatedAt:   time.Now(),
		})
	}
	return campaigns, nil
}

type shopeeCampaignPerfResponse struct {
	shopeeBaseResponse
	Response []struct {
		CampaignList []struct {
			CampaignID json.Number `json:"campaign_id"`
			Metrics    []struct {
				Date       string      `json:"date"`
				Impression json.Number `json:"impression"`
				Clicks     json.Number `json:"clicks"`
				Expense    float64     `json:"expense"`
				DirectGMV  float64     `json:"direct_gmv"`
				BroadGMV   float64     `json:"broad_gmv"`
			} `json:"metrics_list"`
		} `json:"campaign_list"`
	} `json:"response"`
}

func (s *shopeeAPIConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	if len(s.campaignIDs) == 0 {
		return []AdMetric{}, nil
	}
	path := "/api/v2/ads/get_product_campaign_daily_performance"
	query := s.signedQuery(path, url.Values{
		"campaign_id_list": {strings.Join(s.campaignIDs, ",")},
		"start_date":       {s.reportingDay(startDate).Format("2006-01-02")},
		"end_date":         {s.reportingDay(endDate).Format("2006-01-02")},
	})
	var resp shopeeCampaignPerfResponse
	if err := s.client.getJSON(ctx, path, query, nil, &resp); err != nil {
		return nil, err
	}
	if resp.Error != "" && resp.Error != "error_success" {
		return nil, fmt.Errorf("Shopee ads performance API %s: %s", resp.Error, resp.Message)
	}

	var metrics []AdMetric
	for _, shop := range resp.Response {
		for _, campaign := range shop.CampaignList {
			for _, m := range campaign.Metrics {
				date, err := time.Parse("2006-01-02", m.Date)
				if err != nil {
					continue
				}
				impressions, _ := m.Impression.Int64()
				clicks, _ := m.Clicks.Int64()
				metrics = append(metrics, AdMetric{
					CampaignExternalID: campaign.CampaignID.String(),
					Date:               date,
					Impressions:        impressions,
					Clicks:             clicks,
					Spend:              m.Expense,
					// The performance endpoint exposes GMV (direct+broad), not
					// conversion counts; conversions remain 0 until Shopee
					// exposes them per campaign-day.
					Conversions:       0,
					AttributedRevenue: m.DirectGMV + m.BroadGMV,
				})
			}
		}
	}
	return metrics, nil
}

type shopeeOrderListResponse struct {
	shopeeBaseResponse
	Response struct {
		More      bool   `json:"more"`
		NextToken string `json:"next_cursor"`
		OrderList []struct {
			OrderSN     string `json:"order_sn"`
			OrderStatus string `json:"order_status"`
		} `json:"order_list"`
	} `json:"response"`
}

type shopeeOrderDetailResponse struct {
	shopeeBaseResponse
	Response struct {
		OrderList []struct {
			OrderSN     string      `json:"order_sn"`
			TotalAmount float64     `json:"total_amount"`
			CreateTime  json.Number `json:"create_time"`
			OrderStatus string      `json:"order_status"`
		} `json:"order_list"`
	} `json:"response"`
}

// FetchDailySales aggregates orders created in the window: TotalOrders and
// GMV from order detail totals, ReturnedOrders from cancelled/returning
// statuses. Shopee does not report COGS or net sales per order; those columns
// are filled with 0 / GMV.
func (s *shopeeAPIConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	type dayBucket struct {
		orders   int
		gmv      float64
		returned int
	}
	byDay := make(map[string]*dayBucket)
	bucketFor := func(day string) *dayBucket {
		b, ok := byDay[day]
		if !ok {
			b = &dayBucket{}
			byDay[day] = b
		}
		return b
	}

	// 1. Page through every order created in the window.
	var orderSNs []string
	cancelled := make(map[string]bool)
	path := "/api/v2/order/get_order_list"
	// Query and bucketing share the same reporting timezone so near-midnight
	// orders land on the day the seller sees.
	timeFrom := s.reportingDay(startDate).Unix()
	timeTo := s.reportingDay(endDate).Add(24 * time.Hour).Add(-time.Second).Unix()
	cursor := ""
	for {
		extra := url.Values{
			"time_range_field": {"create_time"},
			"time_from":        {strconv.FormatInt(timeFrom, 10)},
			"time_to":          {strconv.FormatInt(timeTo, 10)},
			"page_size":        {"100"},
		}
		if cursor != "" {
			extra.Set("cursor", cursor)
		}
		var resp shopeeOrderListResponse
		if err := s.client.getJSON(ctx, path, s.signedQuery(path, extra), nil, &resp); err != nil {
			return nil, err
		}
		if resp.Error != "" && resp.Error != "error_success" {
			return nil, fmt.Errorf("Shopee order list API %s: %s", resp.Error, resp.Message)
		}
		for _, o := range resp.Response.OrderList {
			orderSNs = append(orderSNs, o.OrderSN)
			switch o.OrderStatus {
			case "CANCELLED", "TO_RETURN", "RETURN_REFUND_IN_PROGRESS":
				cancelled[o.OrderSN] = true
			}
		}
		if !resp.Response.More || resp.Response.NextToken == "" {
			break
		}
		cursor = resp.Response.NextToken
	}

	// 2. Fetch order details in batches of 50 to sum GMV per day.
	const batchSize = 50
	for offset := 0; offset < len(orderSNs); offset += batchSize {
		end := offset + batchSize
		if end > len(orderSNs) {
			end = len(orderSNs)
		}
		batch := orderSNs[offset:end]
		detailPath := "/api/v2/order/get_order_detail"
		query := s.signedQuery(detailPath, url.Values{
			"order_sn_list": {strings.Join(batch, ",")},
		})
		var detail shopeeOrderDetailResponse
		if err := s.client.getJSON(ctx, detailPath, query, nil, &detail); err != nil {
			return nil, err
		}
		if detail.Error != "" && detail.Error != "error_success" {
			return nil, fmt.Errorf("Shopee order detail API %s: %s", detail.Error, detail.Message)
		}
		for _, o := range detail.Response.OrderList {
			createdSecs, err := o.CreateTime.Int64()
			if err != nil {
				continue
			}
			day := time.Unix(createdSecs, 0).In(s.loc).Format("2006-01-02")
			b := bucketFor(day)
			if cancelled[o.OrderSN] {
				b.returned++
				continue
			}
			b.orders++
			b.gmv += o.TotalAmount
		}
	}

	// 3. Materialize one row per day in the window (0-filled days included).
	var metrics []model.DailySalesMetric
	for curr := s.reportingDay(startDate); !curr.After(s.reportingDay(endDate)); curr = curr.AddDate(0, 0, 1) {
		day := curr.Format("2006-01-02")
		b := byDay[day]
		if b == nil {
			metrics = append(metrics, model.DailySalesMetric{Date: curr})
			continue
		}
		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    b.orders,
			GMV:            b.gmv,
			NetSales:       b.gmv,
			COGS:           0,
			ReturnedOrders: b.returned,
		})
	}
	return metrics, nil
}
