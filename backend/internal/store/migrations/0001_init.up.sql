-- Initial schema, equivalent to the legacy inline InitSchema().
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
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_name ON campaigns(name);
CREATE INDEX IF NOT EXISTS idx_daily_ad_metrics_date ON daily_ad_metrics(date);
CREATE INDEX IF NOT EXISTS idx_daily_ad_metrics_campaign_date ON daily_ad_metrics(campaign_id, date);
CREATE INDEX IF NOT EXISTS idx_daily_sales_metrics_date ON daily_sales_metrics(date);
CREATE INDEX IF NOT EXISTS idx_daily_sales_metrics_channel_date ON daily_sales_metrics(channel_id, date);
CREATE INDEX IF NOT EXISTS idx_sync_logs_channel_id ON sync_logs(channel_id);
CREATE INDEX IF NOT EXISTS idx_sync_logs_channel_synced ON sync_logs(channel_id, synced_at);
