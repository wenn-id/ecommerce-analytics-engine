package model

import "time"

type Channel struct {
	ID           int64      `json:"id"`
	Code         string     `json:"code"`
	Name         string     `json:"name"`
	Status       string     `json:"status"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

type Campaign struct {
	ID          int64     `json:"id"`
	ChannelID   int64     `json:"channel_id"`
	ExternalID  string    `json:"external_id"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	DailyBudget float64   `json:"daily_budget"`
	CreatedAt   time.Time `json:"created_at"`
}

type DailyAdMetric struct {
	ID                int64     `json:"id"`
	CampaignID        int64     `json:"campaign_id"`
	Date              time.Time `json:"date"`
	Impressions       int       `json:"impressions"`
	Clicks            int       `json:"clicks"`
	Spend             float64   `json:"spend"`
	Conversions       int       `json:"conversions"`
	AttributedRevenue float64   `json:"attributed_revenue"`
}

type DailySalesMetric struct {
	ID             int64     `json:"id"`
	ChannelID      int64     `json:"channel_id"`
	Date           time.Time `json:"date"`
	TotalOrders    int       `json:"total_orders"`
	GMV            float64   `json:"gmv"`
	NetSales       float64   `json:"net_sales"`
	COGS           float64   `json:"cogs"`
	ReturnedOrders int       `json:"returned_orders"`
}

type SyncLog struct {
	ID               int64     `json:"id"`
	ChannelID        int64     `json:"channel_id"`
	SyncedAt         time.Time `json:"synced_at"`
	Status           string    `json:"status"`
	RecordsProcessed int       `json:"records_processed"`
	ErrorMessage     string    `json:"error_message,omitempty"`
}

type OverviewMetrics struct {
	TotalSpend  float64 `json:"total_spend"`
	TotalGMV    float64 `json:"total_gmv"`
	BlendedROAS float64 `json:"blended_roas"`
	TotalOrders int     `json:"total_orders"`
	AvgCPA      float64 `json:"avg_cpa"`
	ACOS        float64 `json:"acos"`
	NetMargin   float64 `json:"net_margin"`
}

type TrendDataPoint struct {
	Date        string  `json:"date"`
	Spend       float64 `json:"spend"`
	GMV         float64 `json:"gmv"`
	BlendedROAS float64 `json:"blended_roas"`
	TotalOrders int     `json:"total_orders"`
}

type ChannelSummary struct {
	ChannelCode       string  `json:"channel_code"`
	ChannelName       string  `json:"channel_name"`
	TotalSpend        float64 `json:"total_spend"`
	TotalGMV          float64 `json:"total_gmv"`
	ChannelROAS       float64 `json:"channel_roas"`
	SpendSharePercent float64 `json:"spend_share_percent"`
	GMVSharePercent   float64 `json:"gmv_share_percent"`
}

type PaginationMeta struct {
	CurrentPage  int `json:"current_page"`
	Limit        int `json:"limit"`
	TotalRecords int `json:"total_records"`
	TotalPages   int `json:"total_pages"`
}

type PaginatedCampaigns struct {
	Data       []Campaign     `json:"data"`
	Pagination PaginationMeta `json:"pagination"`
}

type CampaignFilter struct {
	ChannelID int64
	Search    string
	Status    string
	Page      int
	Limit     int
}
