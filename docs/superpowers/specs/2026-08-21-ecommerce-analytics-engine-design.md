# E-Commerce & Marketing Analytics Engine - Design Specification

**Date:** 2026-08-21  
**Status:** Approved by Human Partner  
**Architecture Style:** Modular Monolith (Go Backend + Next.js Frontend)

---

## 1. Overview & Objectives

The **E-Commerce & Marketing Analytics Engine** is a full-stack platform designed to aggregate, normalize, and visualize advertising performance and sales data across multiple digital marketing and e-commerce channels (specifically Meta Ads, TikTok Shop, and Shopee).

### Key Goals
- Provide centralized visibility into essential advertising and financial KPIs: Blended ROAS, Channel ROAS, CPA, ACOS, GMV, Ad Spend, Total Orders, and Net Contribution Margin.
- Implement an extensible connector interface to ingest data from disparate platforms via scheduled background synchronization and manual on-demand triggers.
- Deliver a fast, responsive, and type-safe web dashboard built with Next.js, TypeScript, and Tailwind CSS powered by a lightweight, concurrent Go backend.

---

## 2. System Architecture & Directory Structure

The system follows a **Modular Monolith** architecture pattern with clean layer boundaries:

```
ecommerce-analytics/
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go               # Application entry point (HTTP Server, DB init, Scheduler)
│   ├── internal/
│   │   ├── config/                   # Environment and application configuration
│   │   ├── connector/                # Ingestion connector abstraction & platform adapters
│   │   │   ├── connector.go          # PlatformConnector interface & DTO definitions
│   │   │   ├── mock_meta.go          # Meta Ads mock connector implementation
│   │   │   ├── mock_tiktok.go        # TikTok Shop mock connector implementation
│   │   │   └── mock_shopee.go        # Shopee mock connector implementation
│   │   ├── model/                    # Domain models and database entity definitions
│   │   ├── store/                    # Data persistence layer (Repository pattern, SQLite/PostgreSQL)
│   │   │   └── repository.go
│   │   ├── service/                  # Core business logic & analytics computations
│   │   │   ├── analytics_service.go  # Metric aggregation and KPI computation formulas
│   │   │   └── sync_service.go       # Multi-channel sync orchestration
│   │   ├── handler/                  # HTTP REST API handlers & router setup
│   │   │   └── metrics_handler.go
│   │   └── scheduler/                # Background sync scheduler using Go tickers/cron
│   ├── go.mod
│   └── go.sum
└── frontend/
    ├── src/
    │   ├── app/                      # Next.js App Router pages (Dashboard, Settings)
    │   ├── components/               # Reusable UI components (KPI cards, charts, tables)
    │   ├── lib/                      # HTTP client, formatters, and utility functions
    │   └── types/                    # TypeScript type definitions matching backend models
    ├── package.json
    └── tailwind.config.ts
```

---

## 3. Data Models & Unified Schema

### 3.1 Database Tables

1. **`channels`**
   - `id` (INTEGER, PK, Auto Increment)
   - `code` (VARCHAR(50), UNIQUE) — e.g., `'meta_ads'`, `'tiktok_shop'`, `'shopee'`
   - `name` (VARCHAR(100)) — e.g., `'Meta Ads'`, `'TikTok Shop'`, `'Shopee'`
   - `status` (VARCHAR(20)) — `'active'`, `'inactive'`
   - `last_synced_at` (TIMESTAMP, Nullable)

2. **`campaigns`**
   - `id` (INTEGER, PK, Auto Increment)
   - `channel_id` (INTEGER, FK -> `channels.id`)
   - `external_id` (VARCHAR(100)) — Platform-specific campaign ID
   - `name` (VARCHAR(255))
   - `status` (VARCHAR(20)) — `'ACTIVE'`, `'PAUSED'`, `'COMPLETED'`
   - `daily_budget` (DECIMAL(14, 2))
   - `created_at` (TIMESTAMP)

3. **`daily_ad_metrics`**
   - `id` (INTEGER, PK, Auto Increment)
   - `campaign_id` (INTEGER, FK -> `campaigns.id`)
   - `date` (DATE) — `YYYY-MM-DD`
   - `impressions` (INTEGER)
   - `clicks` (INTEGER)
   - `spend` (DECIMAL(14, 2)) — Total advertising cost in IDR / local currency
   - `conversions` (INTEGER) — Attributed purchases / orders
   - `attributed_revenue` (DECIMAL(14, 2)) — Attributed sales revenue
   - *Constraint:* `UNIQUE(campaign_id, date)`

4. **`daily_sales_metrics`**
   - `id` (INTEGER, PK, Auto Increment)
   - `channel_id` (INTEGER, FK -> `channels.id`)
   - `date` (DATE) — `YYYY-MM-DD`
   - `total_orders` (INTEGER)
   - `gmv` (DECIMAL(14, 2)) — Gross Merchandise Value
   - `net_sales` (DECIMAL(14, 2))
   - `cogs` (DECIMAL(14, 2)) — Cost of Goods Sold
   - `returned_orders` (INTEGER)
   - *Constraint:* `UNIQUE(channel_id, date)`

5. **`sync_logs`**
   - `id` (INTEGER, PK, Auto Increment)
   - `channel_id` (INTEGER, FK -> `channels.id`)
   - `synced_at` (TIMESTAMP)
   - `status` (VARCHAR(20)) — `'SUCCESS'`, `'FAILED'`
   - `records_processed` (INTEGER)
   - `error_message` (TEXT, Nullable)

### 3.2 Business Metric Formulas

- **Total Ad Spend:** $\sum \text{daily\_ad\_metrics.spend}$
- **Total GMV:** $\sum \text{daily\_sales\_metrics.gmv}$
- **Total Orders:** $\sum \text{daily\_sales\_metrics.total\_orders}$
- **Blended ROAS:** $\frac{\text{Total GMV}}{\text{Total Ad Spend}}$ (Returns $0.0$ if Spend $= 0$)
- **Channel ROAS:** $\frac{\text{Attributed Revenue for Channel}}{\text{Ad Spend for Channel}}$
- **Average CPA (Cost Per Acquisition):** $\frac{\text{Total Ad Spend}}{\text{Total Orders}}$ (Returns $0.0$ if Orders $= 0$)
- **ACOS (Advertising Cost of Sales):** $\frac{\text{Total Ad Spend}}{\text{Total GMV}} \times 100\%$
- **Net Contribution Margin:** $\text{Total GMV} - \text{Total Ad Spend} - \text{Total COGS} - \text{Estimated Platform Fees (10\%)}$

---

## 4. Connector Engine & Ingestion Pipeline

### 4.1 Connector Abstraction
Every platform implements the Go `PlatformConnector` interface:

```go
type PlatformConnector interface {
    GetChannelCode() string
    FetchCampaigns(ctx context.Context) ([]RawCampaign, error)
    FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]RawAdMetric, error)
    FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]RawSalesMetric, error)
}
```

### 4.2 Ingestion & Concurrency Flow
1. **Trigger:** Can be invoked periodically by the Go scheduler (e.g., hourly/daily) or manually on-demand via `POST /api/v1/sync`.
2. **Concurrent Execution:** `SyncService` spawns a goroutine for each active channel connector to fetch campaign, ad, and sales metrics in parallel.
3. **Idempotent Storage:** Raw records are mapped to domain models and persisted using SQL `UPSERT` statements (`ON CONFLICT (channel_id, date) DO UPDATE ...`) to eliminate data duplication.
4. **Execution Audit:** Outcome metrics (records processed, execution timestamp, errors) are recorded into `sync_logs`.

---

## 5. REST API Specifications

All endpoints return JSON responses under the `/api/v1` namespace.

### Endpoints

1. `GET /api/v1/health`
   - Response: `{"status": "ok", "timestamp": "2026-08-21T07:00:00Z"}`

2. `GET /api/v1/metrics/overview`
   - Query Parameters: `start_date` (ISO date), `end_date` (ISO date)
   - Response:
     ```json
     {
       "total_spend": 15000000.0,
       "total_gmv": 75000000.0,
       "blended_roas": 5.0,
       "total_orders": 350,
       "avg_cpa": 42857.14,
       "acos": 20.0,
       "net_margin": 32500000.0
     }
     ```

3. `GET /api/v1/metrics/trend`
   - Query Parameters: `start_date`, `end_date`, `interval` (`daily` or `weekly`)
   - Response: Array of time-series objects containing `date`, `spend`, `gmv`, `roas`, `orders`.

4. `GET /api/v1/metrics/channels`
   - Query Parameters: `start_date`, `end_date`
   - Response: Array of channel summary objects comparing spend, GMV, and ROAS across Meta, TikTok, and Shopee.

5. `GET /api/v1/campaigns`
   - Query Parameters: `channel_id` (optional), `status` (optional)
   - Response: List of campaigns with individual spend, clicks, conversions, CPA, and ROAS metrics.

6. `POST /api/v1/sync`
   - Request Body: `{"channel_code": "all"}` or specific channel code.
   - Response: `{"status": "sync_completed", "results": [...]}`

7. `GET /api/v1/sync/status`
   - Response: Recent sync logs and current channel statuses.

---

## 6. Frontend Dashboard Architecture (Next.js)

### UI Components
1. **Control Header:**
   - Date range selector preset (Last 7 Days, Last 30 Days, This Month, Custom).
   - "Sync Now" button with loading indicator and last synced timestamp.
2. **KPI Summary Cards:**
   - 5 primary scorecards with delta indicators comparing current period to previous equivalent period.
3. **Dual-Axis Performance Trend Chart:**
   - Combined Bar/Line chart displaying Ad Spend vs GMV bars with an overlay line for Blended ROAS.
4. **Channel Distribution Breakdown:**
   - Donut and horizontal bar charts highlighting Spend Share vs Revenue Share by platform.
5. **Campaign Performance Table:**
   - Searchable, sortable data table detailing campaign status, budget, spend, conversions, CPA, and ROAS.

---

## 7. Error Handling, Validation, & Testing Strategy

### 7.1 Resilience & Error Handling
- **Partial Failure Isolation:** If a single connector encounters an error, other channels continue syncing unaffected.
- **Retry Mechanism:** Connectors implement exponential backoff retry for network transients.
- **Safe Math / Division:** All division operations (ROAS, CPA, ACOS) guard against zero denominators.

### 7.2 Testing Strategy
- **Go Unit Tests:**
  - `analytics_service_test.go`: Validate all business logic formulas (ROAS, CPA, ACOS, Net Margin).
  - `connector_test.go`: Validate mock data integrity and range filters.
- **Go Integration Tests:**
  - `repository_test.go`: Verify idempotent upsert logic and SQL query aggregation.
  - `metrics_handler_test.go`: Verify HTTP routing and JSON serialization using `httptest.ResponseRecorder`.
- **Frontend Type Safety:**
  - Strict TypeScript types mirroring backend DTOs to enforce contract integrity.
