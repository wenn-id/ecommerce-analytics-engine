# E-Commerce & Marketing Analytics Engine Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a full-stack E-Commerce & Marketing Analytics Engine with a concurrent Go backend and a Next.js TypeScript dashboard that ingests, aggregates, and visualizes multi-channel advertising and sales performance.

**Architecture:** Modular Monolith consisting of a Go REST API with pluggable connector interfaces and a concurrent background sync scheduler persisting to SQLite/PostgreSQL, paired with an interactive Next.js App Router frontend.

**Tech Stack:** Go (Golang 1.22+), SQLite (`modernc.org/sqlite` or `github.com/mattn/go-sqlite3`), Next.js 14+ (React, TypeScript, Tailwind CSS, Lucide icons, Recharts).

**Spec:** `docs/superpowers/specs/2026-08-21-ecommerce-analytics-engine-design.md`

## Global Constraints
- All currency financial calculations must be handled as float64 with safe division guarding against division by zero (e.g., zero spend or zero orders).
- Concurrency during multi-channel data ingestion must use Go goroutines with `sync.WaitGroup` and error propagation.
- Database write operations for time-series metrics must be idempotent (`INSERT OR REPLACE` or `ON CONFLICT DO UPDATE`).
- Standard API response envelope must use JSON with consistent date format (`YYYY-MM-DD`).

---

### Task 1: Backend Scaffolding, Configuration, and Domain Models

**Files:**
- Create: `backend/go.mod`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/model/models.go`
- Test: `backend/internal/model/models_test.go`

**Interfaces:**
- Produces:
  - `config.Config` struct with `Port`, `DatabasePath`, `SyncIntervalMinutes`.
  - `model.Channel`, `model.Campaign`, `model.DailyAdMetric`, `model.DailySalesMetric`, `model.SyncLog`.
  - `model.OverviewMetrics`, `model.TrendDataPoint`, `model.ChannelSummary`.

- [ ] **Step 1: Write the failing test for models and serialization**

```go
// backend/internal/model/models_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/model/...`  
Expected: FAIL with package not found or undefined types.

- [ ] **Step 3: Write minimal implementation for models and config**

```go
// backend/internal/model/models.go
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
```

```go
// backend/internal/config/config.go
package config

import "os"

type Config struct {
	Port                string
	DatabasePath        string
	SyncIntervalMinutes int
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "analytics.db"
	}
	return &Config{
		Port:                port,
		DatabasePath:        dbPath,
		SyncIntervalMinutes: 60,
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/model/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/
git commit -m "feat(backend): initialize models and config structure"
```

---

### Task 2: Database Schema & Repository Layer

**Files:**
- Create: `backend/internal/store/db.go`
- Create: `backend/internal/store/repository.go`
- Test: `backend/internal/store/repository_test.go`

**Interfaces:**
- Consumes: `model.Channel`, `model.Campaign`, `model.DailyAdMetric`, `model.DailySalesMetric`, `model.SyncLog`.
- Produces:
  - `store.NewDB(dbPath string) (*sql.DB, error)`
  - `store.Repository` interface with methods:
    - `InitSchema(ctx context.Context) error`
    - `GetChannels(ctx context.Context) ([]model.Channel, error)`
    - `UpsertChannel(ctx context.Context, ch model.Channel) (int64, error)`
    - `UpsertCampaigns(ctx context.Context, campaigns []model.Campaign) error`
    - `UpsertAdMetrics(ctx context.Context, metrics []model.DailyAdMetric) error`
    - `UpsertSalesMetrics(ctx context.Context, metrics []model.DailySalesMetric) error`
    - `InsertSyncLog(ctx context.Context, log model.SyncLog) error`
    - `QueryAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error)`
    - `QuerySalesMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error)`

- [ ] **Step 1: Write the failing test for repository CRUD and upserts**

```go
// backend/internal/store/repository_test.go
package store_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

func TestRepositoryInitAndUpsert(t *testing.T) {
	ctx := context.Background()
	db, err := store.NewDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open test memory db: %v", err)
	}
	defer db.Close()

	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatalf("failed to init schema: %v", err)
	}

	// 1. Insert Channel
	chID, err := repo.UpsertChannel(ctx, model.Channel{
		Code:   "meta_ads",
		Name:   "Meta Ads",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("failed to upsert channel: %v", err)
	}

	// 2. Insert Campaigns
	campaign := model.Campaign{
		ChannelID:   chID,
		ExternalID:  "meta_c_001",
		Name:        "Summer Promo",
		Status:      "ACTIVE",
		DailyBudget: 500000.0,
		CreatedAt:   time.Now(),
	}
	if err := repo.UpsertCampaigns(ctx, []model.Campaign{campaign}); err != nil {
		t.Fatalf("failed to upsert campaign: %v", err)
	}

	// 3. Insert and verify Ad Metrics
	today := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	adMetric := model.DailyAdMetric{
		CampaignID:        1,
		Date:              today,
		Impressions:       5000,
		Clicks:            250,
		Spend:             100000.0,
		Conversions:       10,
		AttributedRevenue: 500000.0,
	}
	if err := repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{adMetric}); err != nil {
		t.Fatalf("failed to upsert ad metric: %v", err)
	}

	metrics, err := repo.QueryAdMetrics(ctx, today, today)
	if err != nil || len(metrics) != 1 {
		t.Fatalf("expected 1 metric, got %d (err: %v)", len(metrics), err)
	}
	if metrics[0].Spend != 100000.0 {
		t.Errorf("expected spend 100000, got %f", metrics[0].Spend)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/store/...`  
Expected: FAIL

- [ ] **Step 3: Write minimal implementation for DB & Repository**

```go
// backend/internal/store/db.go
package store

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func NewDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
```

```go
// backend/internal/store/repository.go
package store

import (
	"context"
	"database/sql"
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
	QuerySalesMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error)
	GetCampaigns(ctx context.Context, channelID int64) ([]model.Campaign, error)
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
		t, _ := time.Parse("2006-01-02", dStr)
		m.Date = t
		results = append(results, m)
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
		t, _ := time.Parse("2006-01-02", dStr)
		m.Date = t
		results = append(results, m)
	}
	return results, nil
}

func (r *sqliteRepository) GetCampaigns(ctx context.Context, channelID int64) ([]model.Campaign, error) {
	var query string
	var args []interface{}
	if channelID > 0 {
		query = "SELECT id, channel_id, external_id, name, status, daily_budget, created_at FROM campaigns WHERE channel_id = ?"
		args = append(args, channelID)
	} else {
		query = "SELECT id, channel_id, external_id, name, status, daily_budget, created_at FROM campaigns"
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var campaigns []model.Campaign
	for rows.Next() {
		var c model.Campaign
		if err := rows.Scan(&c.ID, &c.ChannelID, &c.ExternalID, &c.Name, &c.Status, &c.DailyBudget, &c.CreatedAt); err != nil {
			return nil, err
		}
		campaigns = append(campaigns, c)
	}
	return campaigns, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/store/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/store/
git commit -m "feat(backend): implement sqlite database repository layer"
```

---

### Task 3: Pluggable Connector Interface & Mock Connectors

**Files:**
- Create: `backend/internal/connector/connector.go`
- Create: `backend/internal/connector/mock_meta.go`
- Create: `backend/internal/connector/mock_tiktok.go`
- Create: `backend/internal/connector/mock_shopee.go`
- Test: `backend/internal/connector/connector_test.go`

**Interfaces:**
- Produces:
  - `connector.PlatformConnector` interface.
  - `connector.NewMetaConnector() PlatformConnector`
  - `connector.NewTikTokConnector() PlatformConnector`
  - `connector.NewShopeeConnector() PlatformConnector`

- [ ] **Step 1: Write the failing test for connector data generation**

```go
// backend/internal/connector/connector_test.go
package connector_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
)

func TestConnectors(t *testing.T) {
	ctx := context.Background()
	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	startDate := time.Now().AddDate(0, 0, -7)
	endDate := time.Now()

	for _, conn := range connectors {
		t.Run(conn.GetChannelCode(), func(t *testing.T) {
			campaigns, err := conn.FetchCampaigns(ctx)
			if err != nil || len(campaigns) == 0 {
				t.Fatalf("expected campaigns from %s, got err: %v", conn.GetChannelCode(), err)
			}

			adMetrics, err := conn.FetchDailyAdMetrics(ctx, startDate, endDate)
			if err != nil || len(adMetrics) == 0 {
				t.Fatalf("expected ad metrics from %s, got err: %v", conn.GetChannelCode(), err)
			}

			sales, err := conn.FetchDailySales(ctx, startDate, endDate)
			if err != nil || len(sales) == 0 {
				t.Fatalf("expected sales metrics from %s, got err: %v", conn.GetChannelCode(), err)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/connector/...`  
Expected: FAIL

- [ ] **Step 3: Write minimal implementation for connectors**

```go
// backend/internal/connector/connector.go
package connector

import (
	"context"
	"time"

	"ecommerce-analytics/internal/model"
)

type PlatformConnector interface {
	GetChannelCode() string
	GetChannelName() string
	FetchCampaigns(ctx context.Context) ([]model.Campaign, error)
	FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error)
	FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error)
}
```

```go
// backend/internal/connector/mock_meta.go
package connector

import (
	"context"
	"math/rand"
	"time"

	"ecommerce-analytics/internal/model"
)

type metaConnector struct{}

func NewMetaConnector() PlatformConnector {
	return &metaConnector{}
}

func (m *metaConnector) GetChannelCode() string { return "meta_ads" }
func (m *metaConnector) GetChannelName() string { return "Meta Ads" }

func (m *metaConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "meta_c_101", Name: "Catalog Sales - Retargeting", Status: "ACTIVE", DailyBudget: 750000.0, CreatedAt: time.Now()},
		{ExternalID: "meta_c_102", Name: "Advantage+ Shopping Campaign", Status: "ACTIVE", DailyBudget: 1500000.0, CreatedAt: time.Now()},
	}, nil
}

func (m *metaConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 1200000.0 + float64(rand.Intn(400000))
		clicks := 400 + rand.Intn(150)
		conversions := 25 + rand.Intn(15)
		roas := 4.2 + (rand.Float64() * 1.5)
		attributedRev := spend * roas

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        1,
			Date:              curr,
			Impressions:       18000 + rand.Intn(5000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: attributedRev,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (m *metaConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 30 + rand.Intn(20)
		gmv := float64(orders) * (180000.0 + float64(rand.Intn(40000)))
		cogs := gmv * 0.45
		netSales := gmv * 0.95

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.Intn(3),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}
```

```go
// backend/internal/connector/mock_tiktok.go
package connector

import (
	"context"
	"math/rand"
	"time"

	"ecommerce-analytics/internal/model"
)

type tikTokConnector struct{}

func NewTikTokConnector() PlatformConnector {
	return &tikTokConnector{}
}

func (t *tikTokConnector) GetChannelCode() string { return "tiktok_shop" }
func (t *tikTokConnector) GetChannelName() string { return "TikTok Shop" }

func (t *tikTokConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "tt_c_201", Name: "Product GMV Max - Live Shopping", Status: "ACTIVE", DailyBudget: 1000000.0, CreatedAt: time.Now()},
		{ExternalID: "tt_c_202", Name: "Video Shopping Ads - Top Sellers", Status: "ACTIVE", DailyBudget: 800000.0, CreatedAt: time.Now()},
	}, nil
}

func (t *tikTokConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 900000.0 + float64(rand.Intn(300000))
		clicks := 600 + rand.Intn(200)
		conversions := 35 + rand.Intn(20)
		roas := 4.8 + (rand.Float64() * 1.8)

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        2,
			Date:              curr,
			Impressions:       25000 + rand.Intn(8000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: spend * roas,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (t *tikTokConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 45 + rand.Intn(25)
		gmv := float64(orders) * (150000.0 + float64(rand.Intn(30000)))
		cogs := gmv * 0.42
		netSales := gmv * 0.94

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.Intn(4),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}
```

```go
// backend/internal/connector/mock_shopee.go
package connector

import (
	"context"
	"math/rand"
	"time"

	"ecommerce-analytics/internal/model"
)

type shopeeConnector struct{}

func NewShopeeConnector() PlatformConnector {
	return &shopeeConnector{}
}

func (s *shopeeConnector) GetChannelCode() string { return "shopee" }
func (s *shopeeConnector) GetChannelName() string { return "Shopee" }

func (s *shopeeConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return []model.Campaign{
		{ExternalID: "sh_c_301", Name: "Shopee Discovery Ads - Flash Deals", Status: "ACTIVE", DailyBudget: 600000.0, CreatedAt: time.Now()},
	}, nil
}

func (s *shopeeConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]model.DailyAdMetric, error) {
	var metrics []model.DailyAdMetric
	curr := startDate
	for !curr.After(endDate) {
		spend := 500000.0 + float64(rand.Intn(200000))
		clicks := 300 + rand.Intn(100)
		conversions := 20 + rand.Intn(10)
		roas := 5.2 + (rand.Float64() * 2.0)

		metrics = append(metrics, model.DailyAdMetric{
			CampaignID:        3,
			Date:              curr,
			Impressions:       12000 + rand.Intn(3000),
			Clicks:            clicks,
			Spend:             spend,
			Conversions:       conversions,
			AttributedRevenue: spend * roas,
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}

func (s *shopeeConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	var metrics []model.DailySalesMetric
	curr := startDate
	for !curr.After(endDate) {
		orders := 35 + rand.Intn(15)
		gmv := float64(orders) * (140000.0 + float64(rand.Intn(25000)))
		cogs := gmv * 0.44
		netSales := gmv * 0.93

		metrics = append(metrics, model.DailySalesMetric{
			Date:           curr,
			TotalOrders:    orders,
			GMV:            gmv,
			NetSales:       netSales,
			COGS:           cogs,
			ReturnedOrders: rand.Intn(2),
		})
		curr = curr.AddDate(0, 0, 1)
	}
	return metrics, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/connector/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/connector/
git commit -m "feat(backend): implement platform connectors for Meta, TikTok, and Shopee"
```

---

### Task 4: Analytics Business Logic Service

**Files:**
- Create: `backend/internal/service/analytics_service.go`
- Test: `backend/internal/service/analytics_service_test.go`

**Interfaces:**
- Consumes: `store.Repository`.
- Produces:
  - `service.AnalyticsService` interface:
    - `GetOverviewMetrics(ctx context.Context, start, end time.Time) (*model.OverviewMetrics, error)`
    - `GetTrendData(ctx context.Context, start, end time.Time) ([]model.TrendDataPoint, error)`
    - `GetChannelBreakdown(ctx context.Context, start, end time.Time) ([]model.ChannelSummary, error)`

- [ ] **Step 1: Write the failing test for analytics calculations and zero-guards**

```go
// backend/internal/service/analytics_service_test.go
package service_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestAnalyticsServiceOverview(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	chID, _ := repo.UpsertChannel(ctx, model.Channel{Code: "meta_ads", Name: "Meta Ads", Status: "active"})
	today := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)

	_ = repo.UpsertAdMetrics(ctx, []model.DailyAdMetric{
		{CampaignID: 1, Date: today, Spend: 1000000.0, Conversions: 20, AttributedRevenue: 5000000.0},
	})
	_ = repo.UpsertSalesMetrics(ctx, []model.DailySalesMetric{
		{ChannelID: chID, Date: today, TotalOrders: 25, GMV: 6000000.0, COGS: 2400000.0},
	})

	analyticsSvc := service.NewAnalyticsService(repo)
	overview, err := analyticsSvc.GetOverviewMetrics(ctx, today, today)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if overview.TotalSpend != 1000000.0 || overview.TotalGMV != 6000000.0 {
		t.Errorf("incorrect totals: %+v", overview)
	}
	if overview.BlendedROAS != 6.0 {
		t.Errorf("expected Blended ROAS 6.0, got %f", overview.BlendedROAS)
	}
	if overview.AvgCPA != 40000.0 {
		t.Errorf("expected Avg CPA 40000.0, got %f", overview.AvgCPA)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/service/...`  
Expected: FAIL

- [ ] **Step 3: Write minimal implementation for Analytics Service**

```go
// backend/internal/service/analytics_service.go
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
	for _, s := range salesMetrics {
		totalGMV += s.GMV
		totalCOGS += s.COGS
		totalOrders += s.TotalOrders
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
		// For ad metrics, calculate based on channel campaigns
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/service/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/
git commit -m "feat(backend): implement analytics service business logic formulas"
```

---

### Task 5: Multi-Channel Sync Service & Concurrency Scheduler

**Files:**
- Create: `backend/internal/service/sync_service.go`
- Create: `backend/internal/scheduler/scheduler.go`
- Test: `backend/internal/service/sync_service_test.go`

**Interfaces:**
- Consumes: `connector.PlatformConnector`, `store.Repository`.
- Produces:
  - `service.SyncService` with `SyncAll(ctx context.Context, start, end time.Time) error`
  - `scheduler.StartScheduler(ctx context.Context, syncSvc service.SyncService, intervalMinutes int)`

- [ ] **Step 1: Write the failing test for Sync Service**

```go
// backend/internal/service/sync_service_test.go
package service_test

import (
	"context"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestSyncServiceAll(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)

	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	syncSvc := service.NewSyncService(repo, connectors)
	start := time.Now().AddDate(0, 0, -3)
	end := time.Now()

	if err := syncSvc.SyncAll(ctx, start, end); err != nil {
		t.Fatalf("sync all failed: %v", err)
	}

	channels, _ := repo.GetChannels(ctx)
	if len(channels) != 3 {
		t.Errorf("expected 3 channels registered, got %d", len(channels))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/service/sync_service_test.go`  
Expected: FAIL

- [ ] **Step 3: Write minimal implementation for Sync Service & Scheduler**

```go
// backend/internal/service/sync_service.go
package service

import (
	"context"
	"sync"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/model"
	"ecommerce-analytics/internal/store"
)

type SyncService interface {
	SyncAll(ctx context.Context, start, end time.Time) error
	SyncChannel(ctx context.Context, code string, start, end time.Time) error
}

type syncService struct {
	repo       store.Repository
	connectors map[string]connector.PlatformConnector
}

func NewSyncService(repo store.Repository, conns []connector.PlatformConnector) SyncService {
	connMap := make(map[string]connector.PlatformConnector)
	for _, c := range conns {
		connMap[c.GetChannelCode()] = c
	}
	return &syncService{
		repo:       repo,
		connectors: connMap,
	}
}

func (s *syncService) SyncAll(ctx context.Context, start, end time.Time) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(s.connectors))

	for _, conn := range s.connectors {
		wg.Add(1)
		go func(c connector.PlatformConnector) {
			defer wg.Done()
			if err := s.syncSingle(ctx, c, start, end); err != nil {
				errChan <- err
			}
		}(conn)
	}

	wg.Wait()
	close(errChan)

	if len(errChan) > 0 {
		return <-errChan
	}
	return nil
}

func (s *syncService) SyncChannel(ctx context.Context, code string, start, end time.Time) error {
	conn, ok := s.connectors[code]
	if !ok {
		return nil
	}
	return s.syncSingle(ctx, conn, start, end)
}

func (s *syncService) syncSingle(ctx context.Context, conn connector.PlatformConnector, start, end time.Time) error {
	now := time.Now()
	chID, err := s.repo.UpsertChannel(ctx, model.Channel{
		Code:         conn.GetChannelCode(),
		Name:         conn.GetChannelName(),
		Status:       "active",
		LastSyncedAt: &now,
	})
	if err != nil {
		return err
	}

	campaigns, err := conn.FetchCampaigns(ctx)
	if err == nil {
		for i := range campaigns {
			campaigns[i].ChannelID = chID
		}
		_ = s.repo.UpsertCampaigns(ctx, campaigns)
	}

	adMetrics, err := conn.FetchDailyAdMetrics(ctx, start, end)
	if err == nil {
		for i := range adMetrics {
			adMetrics[i].CampaignID = chID
		}
		_ = s.repo.UpsertAdMetrics(ctx, adMetrics)
	}

	sales, err := conn.FetchDailySales(ctx, start, end)
	if err == nil {
		for i := range sales {
			sales[i].ChannelID = chID
		}
		_ = s.repo.UpsertSalesMetrics(ctx, sales)
	}

	_ = s.repo.InsertSyncLog(ctx, model.SyncLog{
		ChannelID:        chID,
		SyncedAt:         now,
		Status:           "SUCCESS",
		RecordsProcessed: len(adMetrics) + len(sales),
	})

	return nil
}
```

```go
// backend/internal/scheduler/scheduler.go
package scheduler

import (
	"context"
	"time"

	"ecommerce-analytics/internal/service"
)

func Start(ctx context.Context, syncSvc service.SyncService, intervalMinutes int) {
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				end := time.Now()
				start := end.AddDate(0, 0, -30)
				_ = syncSvc.SyncAll(ctx, start, end)
			}
		}
	}()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/service/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/sync_service.go backend/internal/scheduler/
git commit -m "feat(backend): implement multi-channel sync orchestrator and scheduler"
```

---

### Task 6: REST API Handlers & HTTP Server Setup

**Files:**
- Create: `backend/internal/handler/metrics_handler.go`
- Create: `backend/cmd/server/main.go`
- Test: `backend/internal/handler/metrics_handler_test.go`

**Interfaces:**
- Produces:
  - `handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)`
  - HTTP Routes: `/api/v1/health`, `/api/v1/metrics/overview`, `/api/v1/metrics/trend`, `/api/v1/metrics/channels`, `/api/v1/campaigns`, `/api/v1/sync`.

- [ ] **Step 1: Write the failing test for HTTP handlers**

```go
// backend/internal/handler/metrics_handler_test.go
package handler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func TestMetricsHandlerOverview(t *testing.T) {
	ctx := context.Background()
	db, _ := store.NewDB(":memory:")
	defer db.Close()

	repo := store.NewRepository(db)
	_ = repo.InitSchema(ctx)
	conns := []connector.PlatformConnector{connector.NewMetaConnector()}
	syncSvc := service.NewSyncService(repo, conns)
	_ = syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -5), time.Now())

	analyticsSvc := service.NewAnalyticsService(repo)
	h := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)

	req := httptest.NewRequest("GET", "/api/v1/metrics/overview?start_date=2026-08-01&end_date=2026-08-21", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd backend && go test ./internal/handler/...`  
Expected: FAIL

- [ ] **Step 3: Write minimal implementation for HTTP Handler and Server Main**

```go
// backend/internal/handler/metrics_handler.go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

type MetricsHandler struct {
	mux          *http.ServeMux
	analyticsSvc service.AnalyticsService
	syncSvc      service.SyncService
	repo         store.Repository
}

func NewMetricsHandler(analyticsSvc service.AnalyticsService, syncSvc service.SyncService, repo store.Repository) *MetricsHandler {
	h := &MetricsHandler{
		mux:          http.NewServeMux(),
		analyticsSvc: analyticsSvc,
		syncSvc:      syncSvc,
		repo:         repo,
	}
	h.registerRoutes()
	return h
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for Next.js dev server
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	h.mux.ServeHTTP(w, r)
}

func (h *MetricsHandler) registerRoutes() {
	h.mux.HandleFunc("GET /api/v1/health", h.handleHealth)
	h.mux.HandleFunc("GET /api/v1/metrics/overview", h.handleOverview)
	h.mux.HandleFunc("GET /api/v1/metrics/trend", h.handleTrend)
	h.mux.HandleFunc("GET /api/v1/metrics/channels", h.handleChannels)
	h.mux.HandleFunc("GET /api/v1/campaigns", h.handleCampaigns)
	h.mux.HandleFunc("POST /api/v1/sync", h.handleSync)
}

func (h *MetricsHandler) parseDateRange(r *http.Request) (time.Time, time.Time) {
	startStr := r.URL.Query().Get("start_date")
	endStr := r.URL.Query().Get("end_date")

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		end = time.Now()
	}
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		start = end.AddDate(0, 0, -30)
	}
	return start, end
}

func (h *MetricsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, http.StatusOK, map[string]string{"status": "ok", "timestamp": time.Now().Format(time.RFC3339)})
}

func (h *MetricsHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetOverviewMetrics(r.Context(), start, end)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleTrend(w http.ResponseWriter, r *http.Request) {
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetTrendData(r.Context(), start, end)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleChannels(w http.ResponseWriter, r *http.Request) {
	start, end := h.parseDateRange(r)
	data, err := h.analyticsSvc.GetChannelBreakdown(r.Context(), start, end)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleCampaigns(w http.ResponseWriter, r *http.Request) {
	chIDStr := r.URL.Query().Get("channel_id")
	var chID int64
	if chIDStr != "" {
		chID, _ = strconv.ParseInt(chIDStr, 10, 64)
	}
	data, err := h.repo.GetCampaigns(r.Context(), chID)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, data)
}

func (h *MetricsHandler) handleSync(w http.ResponseWriter, r *http.Request) {
	end := time.Now()
	start := end.AddDate(0, 0, -30)
	if err := h.syncSvc.SyncAll(r.Context(), start, end); err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]string{"status": "sync_completed"})
}

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}
```

```go
// backend/cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"ecommerce-analytics/internal/config"
	"ecommerce-analytics/internal/connector"
	"ecommerce-analytics/internal/handler"
	"ecommerce-analytics/internal/scheduler"
	"ecommerce-analytics/internal/service"
	"ecommerce-analytics/internal/store"
)

func main() {
	cfg := config.Load()

	db, err := store.NewDB(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	repo := store.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		log.Fatalf("failed to init schema: %v", err)
	}

	connectors := []connector.PlatformConnector{
		connector.NewMetaConnector(),
		connector.NewTikTokConnector(),
		connector.NewShopeeConnector(),
	}

	syncSvc := service.NewSyncService(repo, connectors)
	analyticsSvc := service.NewAnalyticsService(repo)

	// Perform initial sync
	log.Println("Performing initial multi-channel sync...")
	_ = syncSvc.SyncAll(ctx, time.Now().AddDate(0, 0, -30), time.Now())

	// Start background scheduler
	scheduler.Start(ctx, syncSvc, cfg.SyncIntervalMinutes)

	apiHandler := handler.NewMetricsHandler(analyticsSvc, syncSvc, repo)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: apiHandler,
	}

	log.Printf("Analytics Engine Server listening on port %s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd backend && go test ./internal/handler/... -v`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/handler/ backend/cmd/server/
git commit -m "feat(backend): implement REST API endpoints and server entrypoint"
```

---

### Task 7: Frontend Next.js Project Scaffolding & API Client

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/src/types/analytics.ts`
- Create: `frontend/src/lib/api.ts`
- Create: `frontend/src/lib/utils.ts`

**Interfaces:**
- Produces:
  - TypeScript types: `OverviewMetrics`, `TrendDataPoint`, `ChannelSummary`, `Campaign`.
  - API functions: `fetchOverview()`, `fetchTrend()`, `fetchChannels()`, `fetchCampaigns()`, `triggerSync()`.

- [ ] **Step 1: Scaffold Next.js Package and Types**

```json
// frontend/package.json
{
  "name": "ecommerce-analytics-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint"
  },
  "dependencies": {
    "clsx": "^2.1.1",
    "lucide-react": "^0.395.0",
    "next": "^14.2.4",
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "recharts": "^2.12.7",
    "tailwind-merge": "^2.3.0"
  },
  "devDependencies": {
    "@types/node": "^20",
    "@types/react": "^18",
    "@types/react-dom": "^18",
    "postcss": "^8",
    "tailwindcss": "^3.4.1",
    "typescript": "^5"
  }
}
```

```typescript
// frontend/src/types/analytics.ts
export interface OverviewMetrics {
  total_spend: number;
  total_gmv: number;
  blended_roas: number;
  total_orders: number;
  avg_cpa: number;
  acos: number;
  net_margin: number;
}

export interface TrendDataPoint {
  date: string;
  spend: number;
  gmv: number;
  blended_roas: number;
  total_orders: number;
}

export interface ChannelSummary {
  channel_code: string;
  channel_name: string;
  total_spend: number;
  total_gmv: number;
  channel_roas: number;
  spend_share_percent: number;
  gmv_share_percent: number;
}

export interface Campaign {
  id: number;
  channel_id: number;
  external_id: string;
  name: string;
  status: string;
  daily_budget: number;
  created_at: string;
}
```

- [ ] **Step 2: Implement API Client**

```typescript
// frontend/src/lib/api.ts
import { OverviewMetrics, TrendDataPoint, ChannelSummary, Campaign } from '../types/analytics';

const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api/v1';

export async function fetchOverview(startDate: string, endDate: string): Promise<OverviewMetrics> {
  const res = await fetch(`${API_BASE}/metrics/overview?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch overview metrics');
  return res.json();
}

export async function fetchTrend(startDate: string, endDate: string): Promise<TrendDataPoint[]> {
  const res = await fetch(`${API_BASE}/metrics/trend?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch trend data');
  return res.json();
}

export async function fetchChannels(startDate: string, endDate: string): Promise<ChannelSummary[]> {
  const res = await fetch(`${API_BASE}/metrics/channels?start_date=${startDate}&end_date=${endDate}`);
  if (!res.ok) throw new Error('Failed to fetch channel breakdown');
  return res.json();
}

export async function fetchCampaigns(): Promise<Campaign[]> {
  const res = await fetch(`${API_BASE}/campaigns`);
  if (!res.ok) throw new Error('Failed to fetch campaigns');
  return res.json();
}

export async function triggerSync(): Promise<{ status: string }> {
  const res = await fetch(`${API_BASE}/sync`, { method: 'POST' });
  if (!res.ok) throw new Error('Failed to trigger sync');
  return res.json();
}
```

- [ ] **Step 3: Verify build / type definitions**

Run: `cd frontend && npx tsc --noEmit`  
Expected: PASS (0 errors)

- [ ] **Step 4: Commit**

```bash
git add frontend/
git commit -m "feat(frontend): setup types, package config, and REST API client"
```

---

### Task 8: Frontend Dashboard UI (KPI Cards, Charts, & Data Tables)

**Files:**
- Create: `frontend/src/components/MetricCard.tsx`
- Create: `frontend/src/components/TrendChart.tsx`
- Create: `frontend/src/components/ChannelBreakdown.tsx`
- Create: `frontend/src/components/CampaignTable.tsx`
- Create: `frontend/src/app/page.tsx`

**Interfaces:**
- Produces: Complete interactive Dashboard view with real-time sync trigger and date range filter.

- [ ] **Step 1: Implement Metric Scorecard Component**

```tsx
// frontend/src/components/MetricCard.tsx
import React from 'react';
import { LucideIcon } from 'lucide-react';

interface MetricCardProps {
  title: string;
  value: string;
  subtitle?: string;
  icon: LucideIcon;
  trend?: string;
  trendPositive?: boolean;
}

export const MetricCard: React.FC<MetricCardProps> = ({
  title,
  value,
  subtitle,
  icon: Icon,
  trend,
  trendPositive = true,
}) => {
  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm flex flex-col justify-between">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-gray-500">{title}</span>
        <div className="p-2 bg-indigo-50 text-indigo-600 rounded-lg">
          <Icon className="w-5 h-5" />
        </div>
      </div>
      <div className="mt-4">
        <h3 className="text-2xl font-bold text-gray-900">{value}</h3>
        {subtitle && <p className="text-xs text-gray-400 mt-1">{subtitle}</p>}
        {trend && (
          <span className={`inline-block text-xs font-semibold mt-2 px-2 py-0.5 rounded ${
            trendPositive ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'
          }`}>
            {trend}
          </span>
        )}
      </div>
    </div>
  );
};
```

- [ ] **Step 2: Implement Performance Trend Chart Component**

```tsx
// frontend/src/components/TrendChart.tsx
'use client';
import React from 'react';
import {
  ResponsiveContainer,
  ComposedChart,
  Bar,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  Legend,
  CartesianGrid,
} from 'recharts';
import { TrendDataPoint } from '../types/analytics';

export const TrendChart: React.FC<{ data: TrendDataPoint[] }> = ({ data }) => {
  return (
    <div className="bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
      <h3 className="text-lg font-bold text-gray-900 mb-4">Ad Spend vs GMV & Blended ROAS Trend</h3>
      <div className="h-80 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <ComposedChart data={data} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" tick={{ fontSize: 12 }} />
            <YAxis yAxisId="left" tickFormatter={(v) => `Rp ${(v / 1000000).toFixed(1)}M`} />
            <YAxis yAxisId="right" orientation="right" tickFormatter={(v) => `${v}x`} />
            <Tooltip
              formatter={(val: number, name: string) =>
                name === 'Blended ROAS'
                  ? [`${val.toFixed(2)}x`, name]
                  : [`Rp ${val.toLocaleString('id-ID')}`, name]
              }
            />
            <Legend />
            <Bar yAxisId="left" dataKey="spend" name="Ad Spend" fill="#ef4444" radius={[4, 4, 0, 0]} />
            <Bar yAxisId="left" dataKey="gmv" name="GMV" fill="#3b82f6" radius={[4, 4, 0, 0]} />
            <Line yAxisId="right" type="monotone" dataKey="blended_roas" name="Blended ROAS" stroke="#10b981" strokeWidth={3} dot={{ r: 3 }} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
};
```

- [ ] **Step 3: Implement Dashboard Page with Data Ingestion Controls**

```tsx
// frontend/src/app/page.tsx
'use client';
import React, { useState, useEffect } from 'react';
import { DollarSign, ShoppingCart, TrendingUp, Percent, RefreshCw, Layers } from 'lucide-react';
import { fetchOverview, fetchTrend, fetchChannels, fetchCampaigns, triggerSync } from '../lib/api';
import { OverviewMetrics, TrendDataPoint, ChannelSummary, Campaign } from '../types/analytics';
import { MetricCard } from '../components/MetricCard';
import { TrendChart } from '../components/TrendChart';

export default function DashboardPage() {
  const [overview, setOverview] = useState<OverviewMetrics | null>(null);
  const [trend, setTrend] = useState<TrendDataPoint[]>([]);
  const [channels, setChannels] = useState<ChannelSummary[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [syncing, setSyncing] = useState(false);

  const loadData = async () => {
    const start = '2026-08-01';
    const end = '2026-08-21';
    try {
      const [ov, tr, ch, camp] = await Promise.all([
        fetchOverview(start, end),
        fetchTrend(start, end),
        fetchChannels(start, end),
        fetchCampaigns(),
      ]);
      setOverview(ov);
      setTrend(tr);
      setChannels(ch);
      setCampaigns(camp);
    } catch (err) {
      console.error(err);
    }
  };

  useEffect(() => {
    loadData();
  }, []);

  const handleSync = async () => {
    setSyncing(true);
    try {
      await triggerSync();
      await loadData();
    } finally {
      setSyncing(false);
    }
  };

  return (
    <main className="min-h-screen bg-gray-50 p-8">
      <div className="max-w-7xl mx-auto space-y-8">
        <header className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-extrabold text-gray-900">E-Commerce & Ads Analytics Engine</h1>
            <p className="text-gray-500 mt-1">Multi-Channel Performance Overview (Meta Ads, TikTok Shop, Shopee)</p>
          </div>
          <button
            onClick={handleSync}
            disabled={syncing}
            className="flex items-center gap-2 bg-indigo-600 hover:bg-indigo-700 text-white px-5 py-2.5 rounded-lg font-medium shadow-sm transition disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${syncing ? 'animate-spin' : ''}`} />
            {syncing ? 'Syncing...' : 'Sync Now'}
          </button>
        </header>

        {overview && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
            <MetricCard title="Total Ad Spend" value={`Rp ${(overview.total_spend / 1000000).toFixed(1)}M`} icon={DollarSign} />
            <MetricCard title="Total GMV" value={`Rp ${(overview.total_gmv / 1000000).toFixed(1)}M`} icon={ShoppingCart} />
            <MetricCard title="Blended ROAS" value={`${overview.blended_roas}x`} icon={TrendingUp} trendPositive={overview.blended_roas >= 4} />
            <MetricCard title="Average CPA" value={`Rp ${overview.avg_cpa.toLocaleString('id-ID')}`} icon={Percent} />
            <MetricCard title="Net Contribution Margin" value={`Rp ${(overview.net_margin / 1000000).toFixed(1)}M`} icon={Layers} />
          </div>
        )}

        <TrendChart data={trend} />

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1 bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
            <h3 className="text-lg font-bold text-gray-900 mb-4">Channel Performance Breakdown</h3>
            <div className="space-y-4">
              {channels.map((ch) => (
                <div key={ch.channel_code} className="p-4 border rounded-lg flex justify-between items-center">
                  <div>
                    <h4 className="font-semibold text-gray-800">{ch.channel_name}</h4>
                    <p className="text-xs text-gray-500">Spend: Rp {(ch.total_spend / 1000000).toFixed(1)}M</p>
                  </div>
                  <div className="text-right">
                    <span className="text-sm font-bold text-indigo-600">{ch.channel_roas}x ROAS</span>
                    <p className="text-xs text-gray-400">GMV: Rp {(ch.total_gmv / 1000000).toFixed(1)}M</p>
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="lg:col-span-2 bg-white p-6 rounded-xl border border-gray-100 shadow-sm">
            <h3 className="text-lg font-bold text-gray-900 mb-4">Active Campaigns</h3>
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b text-gray-400">
                    <th className="pb-3">Campaign Name</th>
                    <th className="pb-3">Status</th>
                    <th className="pb-3 text-right">Daily Budget</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {campaigns.map((c) => (
                    <tr key={c.id} className="text-gray-700">
                      <td className="py-3 font-medium">{c.name}</td>
                      <td className="py-3"><span className="px-2 py-1 bg-green-50 text-green-700 rounded-full text-xs">{c.status}</span></td>
                      <td className="py-3 text-right font-semibold">Rp {c.daily_budget.toLocaleString('id-ID')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/
git commit -m "feat(frontend): implement dashboard KPI cards, trend charts, and campaign table"
```
