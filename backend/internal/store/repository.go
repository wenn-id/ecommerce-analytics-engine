package store

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"ecommerce-analytics/internal/model"
)

type Repository interface {
	InitSchema(ctx context.Context) error
	GetChannels(ctx context.Context) ([]model.Channel, error)
	UpsertChannel(ctx context.Context, ch model.Channel) (int64, error)
	UpsertCampaigns(ctx context.Context, campaigns []model.Campaign) error
	UpsertAdMetrics(ctx context.Context, metrics []model.DailyAdMetric) error
	UpsertSalesMetrics(ctx context.Context, metrics []model.DailySalesMetric) error
	InsertSyncLog(ctx context.Context, log model.SyncLog) error
	QueryAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error)
	GetCampaigns(ctx context.Context, filter model.CampaignFilter) ([]model.Campaign, int, error)
	GetAggregatedOverview(ctx context.Context, startDate, endDate time.Time) (totalSpend float64, totalGMV float64, totalCOGS float64, totalOrders int, err error)
	GetAggregatedDailyTrends(ctx context.Context, startDate, endDate time.Time) ([]model.TrendDataPoint, error)
	GetAggregatedChannelSummaries(ctx context.Context, startDate, endDate time.Time) ([]model.ChannelSummary, error)
}

type sqliteRepository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &sqliteRepository{db: db}
}

func (r *sqliteRepository) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		last_synced_at DATETIME
	);

	CREATE TABLE IF NOT EXISTS campaigns (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		external_id TEXT NOT NULL,
		name TEXT NOT NULL,
		status TEXT NOT NULL,
		daily_budget REAL NOT NULL,
		created_at DATETIME NOT NULL,
		UNIQUE(channel_id, external_id)
	);

	CREATE TABLE IF NOT EXISTS daily_ad_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		campaign_id INTEGER NOT NULL,
		date DATE NOT NULL,
		impressions INTEGER NOT NULL,
		clicks INTEGER NOT NULL,
		spend REAL NOT NULL,
		conversions INTEGER NOT NULL,
		attributed_revenue REAL NOT NULL,
		UNIQUE(campaign_id, date)
	);

	CREATE TABLE IF NOT EXISTS daily_sales_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		date DATE NOT NULL,
		total_orders INTEGER NOT NULL,
		gmv REAL NOT NULL,
		net_sales REAL NOT NULL,
		cogs REAL NOT NULL,
		returned_orders INTEGER NOT NULL,
		UNIQUE(channel_id, date)
	);

	CREATE TABLE IF NOT EXISTS sync_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL,
		synced_at DATETIME NOT NULL,
		status TEXT NOT NULL,
		records_processed INTEGER NOT NULL,
		error_message TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_campaigns_channel_id ON campaigns(channel_id);
	CREATE INDEX IF NOT EXISTS idx_daily_ad_metrics_date ON daily_ad_metrics(date);
	CREATE INDEX IF NOT EXISTS idx_daily_ad_metrics_campaign_date ON daily_ad_metrics(campaign_id, date);
	CREATE INDEX IF NOT EXISTS idx_daily_sales_metrics_date ON daily_sales_metrics(date);
	CREATE INDEX IF NOT EXISTS idx_daily_sales_metrics_channel_date ON daily_sales_metrics(channel_id, date);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_channel_id ON sync_logs(channel_id);
	`
	_, err := r.db.ExecContext(ctx, schema)
	return err
}

func (r *sqliteRepository) UpsertChannel(ctx context.Context, ch model.Channel) (int64, error) {
	query := `
	INSERT INTO channels (code, name, status, last_synced_at)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(code) DO UPDATE SET
		name=excluded.name,
		status=excluded.status,
		last_synced_at=excluded.last_synced_at;
	`
	res, err := r.db.ExecContext(ctx, query, ch.Code, ch.Name, ch.Status, ch.LastSyncedAt)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil || id == 0 {
		var existingID int64
		err = r.db.QueryRowContext(ctx, "SELECT id FROM channels WHERE code = ?", ch.Code).Scan(&existingID)
		return existingID, err
	}
	return id, nil
}

func (r *sqliteRepository) GetChannels(ctx context.Context) ([]model.Channel, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, code, name, status, last_synced_at FROM channels")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []model.Channel
	for rows.Next() {
		var ch model.Channel
		var lastSync sql.NullTime
		if err := rows.Scan(&ch.ID, &ch.Code, &ch.Name, &ch.Status, &lastSync); err != nil {
			return nil, err
		}
		if lastSync.Valid {
			ch.LastSyncedAt = &lastSync.Time
		}
		channels = append(channels, ch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *sqliteRepository) UpsertCampaigns(ctx context.Context, campaigns []model.Campaign) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO campaigns (channel_id, external_id, name, status, daily_budget, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(channel_id, external_id) DO UPDATE SET
			name=excluded.name,
			status=excluded.status,
			daily_budget=excluded.daily_budget;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, c := range campaigns {
		if _, err := stmt.ExecContext(ctx, c.ChannelID, c.ExternalID, c.Name, c.Status, c.DailyBudget, c.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) UpsertAdMetrics(ctx context.Context, metrics []model.DailyAdMetric) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO daily_ad_metrics (campaign_id, date, impressions, clicks, spend, conversions, attributed_revenue)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(campaign_id, date) DO UPDATE SET
			impressions=excluded.impressions,
			clicks=excluded.clicks,
			spend=excluded.spend,
			conversions=excluded.conversions,
			attributed_revenue=excluded.attributed_revenue;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		dateStr := m.Date.Format("2006-01-02")
		if _, err := stmt.ExecContext(ctx, m.CampaignID, dateStr, m.Impressions, m.Clicks, m.Spend, m.Conversions, m.AttributedRevenue); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) UpsertSalesMetrics(ctx context.Context, metrics []model.DailySalesMetric) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO daily_sales_metrics (channel_id, date, total_orders, gmv, net_sales, cogs, returned_orders)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(channel_id, date) DO UPDATE SET
			total_orders=excluded.total_orders,
			gmv=excluded.gmv,
			net_sales=excluded.net_sales,
			cogs=excluded.cogs,
			returned_orders=excluded.returned_orders;
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range metrics {
		dateStr := m.Date.Format("2006-01-02")
		if _, err := stmt.ExecContext(ctx, m.ChannelID, dateStr, m.TotalOrders, m.GMV, m.NetSales, m.COGS, m.ReturnedOrders); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sqliteRepository) InsertSyncLog(ctx context.Context, log model.SyncLog) error {
	query := `
	INSERT INTO sync_logs (channel_id, synced_at, status, records_processed, error_message)
	VALUES (?, ?, ?, ?, ?);
	`
	_, err := r.db.ExecContext(ctx, query, log.ChannelID, log.SyncedAt, log.Status, log.RecordsProcessed, log.ErrorMessage)
	return err
}

func (r *sqliteRepository) QueryAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	query := `
	SELECT id, campaign_id, date, impressions, clicks, spend, conversions, attributed_revenue
	FROM daily_ad_metrics
	WHERE date >= ? AND date <= ?
	ORDER BY date ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.DailyAdMetric
	for rows.Next() {
		var m model.DailyAdMetric
		var dStr string
		if err := rows.Scan(&m.ID, &m.CampaignID, &dStr, &m.Impressions, &m.Clicks, &m.Spend, &m.Conversions, &m.AttributedRevenue); err != nil {
			return nil, err
		}
		t, err := parseDate(dStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format in daily_ad_metrics %q: %w", dStr, err)
		}
		m.Date = t
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func (r *sqliteRepository) QuerySalesMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	query := `
	SELECT id, channel_id, date, total_orders, gmv, net_sales, cogs, returned_orders
	FROM daily_sales_metrics
	WHERE date >= ? AND date <= ?
	ORDER BY date ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []model.DailySalesMetric
	for rows.Next() {
		var m model.DailySalesMetric
		var dStr string
		if err := rows.Scan(&m.ID, &m.ChannelID, &dStr, &m.TotalOrders, &m.GMV, &m.NetSales, &m.COGS, &m.ReturnedOrders); err != nil {
			return nil, err
		}
		t, err := parseDate(dStr)
		if err != nil {
			return nil, fmt.Errorf("invalid date format in daily_sales_metrics %q: %w", dStr, err)
		}
		m.Date = t
		results = append(results, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

func parseDate(dStr string) (time.Time, error) {
	if len(dStr) >= 10 {
		return time.Parse("2006-01-02", dStr[:10])
	}
	return time.Parse("2006-01-02", dStr)
}

func (r *sqliteRepository) GetCampaigns(ctx context.Context, filter model.CampaignFilter) ([]model.Campaign, int, error) {
	whereClauses := []string{"1=1"}
	var args []interface{}

	if filter.ChannelID > 0 {
		whereClauses = append(whereClauses, "channel_id = ?")
		args = append(args, filter.ChannelID)
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, "status = ?")
		args = append(args, filter.Status)
	}

	if filter.Search != "" {
		whereClauses = append(whereClauses, "(name LIKE ? OR external_id LIKE ?)")
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := "SELECT COUNT(*) FROM campaigns WHERE " + whereSQL
	var totalRecords int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalRecords); err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit

	query := fmt.Sprintf("SELECT id, channel_id, external_id, name, status, daily_budget, created_at FROM campaigns WHERE %s ORDER BY id ASC LIMIT ? OFFSET ?", whereSQL)
	queryArgs := append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var campaigns []model.Campaign
	for rows.Next() {
		var c model.Campaign
		if err := rows.Scan(&c.ID, &c.ChannelID, &c.ExternalID, &c.Name, &c.Status, &c.DailyBudget, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		campaigns = append(campaigns, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if campaigns == nil {
		campaigns = []model.Campaign{}
	}

	return campaigns, totalRecords, nil
}

func (r *sqliteRepository) GetAggregatedOverview(ctx context.Context, startDate, endDate time.Time) (float64, float64, float64, int, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	var totalSpend float64
	spendQuery := "SELECT COALESCE(SUM(spend), 0) FROM daily_ad_metrics WHERE date >= ? AND date <= ?"
	if err := r.db.QueryRowContext(ctx, spendQuery, startStr, endStr).Scan(&totalSpend); err != nil {
		return 0, 0, 0, 0, err
	}

	var totalGMV, totalCOGS float64
	var totalOrders int
	salesQuery := "SELECT COALESCE(SUM(gmv), 0), COALESCE(SUM(cogs), 0), COALESCE(SUM(total_orders), 0) FROM daily_sales_metrics WHERE date >= ? AND date <= ?"
	if err := r.db.QueryRowContext(ctx, salesQuery, startStr, endStr).Scan(&totalGMV, &totalCOGS, &totalOrders); err != nil {
		return 0, 0, 0, 0, err
	}

	return totalSpend, totalGMV, totalCOGS, totalOrders, nil
}

func (r *sqliteRepository) GetAggregatedDailyTrends(ctx context.Context, startDate, endDate time.Time) ([]model.TrendDataPoint, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	query := `
	SELECT 
		d.date,
		COALESCE(ad.total_spend, 0) as spend,
		COALESCE(s.total_gmv, 0) as gmv,
		COALESCE(s.total_orders, 0) as total_orders
	FROM (
		SELECT date FROM daily_ad_metrics WHERE date >= ? AND date <= ?
		UNION
		SELECT date FROM daily_sales_metrics WHERE date >= ? AND date <= ?
	) d
	LEFT JOIN (
		SELECT date, SUM(spend) as total_spend 
		FROM daily_ad_metrics 
		WHERE date >= ? AND date <= ? 
		GROUP BY date
	) ad ON d.date = ad.date
	LEFT JOIN (
		SELECT date, SUM(gmv) as total_gmv, SUM(total_orders) as total_orders 
		FROM daily_sales_metrics 
		WHERE date >= ? AND date <= ? 
		GROUP BY date
	) s ON d.date = s.date
	ORDER BY d.date ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, startStr, endStr, startStr, endStr, startStr, endStr, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dataMap := make(map[string]model.TrendDataPoint)
	for rows.Next() {
		var dStr string
		var pt model.TrendDataPoint
		if err := rows.Scan(&dStr, &pt.Spend, &pt.GMV, &pt.TotalOrders); err != nil {
			return nil, err
		}
		t, err := parseDate(dStr)
		if err == nil {
			dStr = t.Format("2006-01-02")
		}
		pt.Date = dStr
		dataMap[dStr] = pt
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var results []model.TrendDataPoint
	curr := startDate
	for !curr.After(endDate) {
		dStr := curr.Format("2006-01-02")
		pt, ok := dataMap[dStr]
		if !ok {
			pt = model.TrendDataPoint{Date: dStr}
		}
		if pt.Spend > 0 {
			pt.BlendedROAS = math.Round((pt.GMV/pt.Spend)*100) / 100
		}
		pt.Spend = math.Round(pt.Spend*100) / 100
		pt.GMV = math.Round(pt.GMV*100) / 100
		results = append(results, pt)
		curr = curr.AddDate(0, 0, 1)
	}

	return results, nil
}

func (r *sqliteRepository) GetAggregatedChannelSummaries(ctx context.Context, startDate, endDate time.Time) ([]model.ChannelSummary, error) {
	startStr := startDate.Format("2006-01-02")
	endStr := endDate.Format("2006-01-02")

	// Query overall total spend and total GMV across entire dataset for consistent denominators
	var totalSpend, totalGMV float64
	spendQuery := "SELECT COALESCE(SUM(spend), 0) FROM daily_ad_metrics WHERE date >= ? AND date <= ?"
	if err := r.db.QueryRowContext(ctx, spendQuery, startStr, endStr).Scan(&totalSpend); err != nil {
		return nil, err
	}
	salesQuery := "SELECT COALESCE(SUM(gmv), 0) FROM daily_sales_metrics WHERE date >= ? AND date <= ?"
	if err := r.db.QueryRowContext(ctx, salesQuery, startStr, endStr).Scan(&totalGMV); err != nil {
		return nil, err
	}

	query := `
	SELECT 
		c.code,
		c.name,
		COALESCE(ad.total_spend, 0) as total_spend,
		COALESCE(s.total_gmv, 0) as total_gmv
	FROM channels c
	LEFT JOIN (
		SELECT cmp.channel_id, SUM(m.spend) as total_spend
		FROM daily_ad_metrics m
		JOIN campaigns cmp ON m.campaign_id = cmp.id
		WHERE m.date >= ? AND m.date <= ?
		GROUP BY cmp.channel_id
	) ad ON c.id = ad.channel_id
	LEFT JOIN (
		SELECT channel_id, SUM(gmv) as total_gmv
		FROM daily_sales_metrics
		WHERE date >= ? AND date <= ?
		GROUP BY channel_id
	) s ON c.id = s.channel_id
	ORDER BY c.id ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, startStr, endStr, startStr, endStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type channelData struct {
		code  string
		name  string
		spend float64
		gmv   float64
	}

	var channelsData []channelData
	for rows.Next() {
		var cd channelData
		if err := rows.Scan(&cd.code, &cd.name, &cd.spend, &cd.gmv); err != nil {
			return nil, err
		}
		channelsData = append(channelsData, cd)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var summaries []model.ChannelSummary
	for _, cd := range channelsData {
		var roas float64
		if cd.spend > 0 {
			roas = cd.gmv / cd.spend
		}
		summaries = append(summaries, model.ChannelSummary{
			ChannelCode:       cd.code,
			ChannelName:       cd.name,
			TotalSpend:        math.Round(cd.spend*100) / 100,
			TotalGMV:          math.Round(cd.gmv*100) / 100,
			ChannelROAS:       math.Round(roas*100) / 100,
			SpendSharePercent: math.Round((cd.spend/math.Max(totalSpend, 1.0))*10000) / 100,
			GMVSharePercent:   math.Round((cd.gmv/math.Max(totalGMV, 1.0))*10000) / 100,
		})
	}
	return summaries, nil
}
