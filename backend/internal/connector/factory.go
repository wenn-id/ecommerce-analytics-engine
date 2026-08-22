package connector

import (
	"context"
	"fmt"
	"os"
	"time"

	"ecommerce-analytics/internal/model"
)

// EnvSource abstracts environment lookup so tests can drive the factory
// without mutating the process environment.
type EnvSource func(string) string

func osGetenv(key string) string { return os.Getenv(key) }

// unavailableConnector occupies a channel slot in production deployments
// whose credentials are missing. Every fetch fails loudly with the reason, so
// a missing platform surfaces as a FAILED sync log instead of silently
// persisting synthetic mock data (#62, review: never write mock data in
// production).
type unavailableConnector struct {
	code   string
	name   string
	reason error
}

func (u *unavailableConnector) GetChannelCode() string { return u.code }
func (u *unavailableConnector) GetChannelName() string { return u.name }

func (u *unavailableConnector) FetchCampaigns(ctx context.Context) ([]model.Campaign, error) {
	return nil, u.fail()
}

func (u *unavailableConnector) FetchDailyAdMetrics(ctx context.Context, startDate, endDate time.Time) ([]AdMetric, error) {
	return nil, u.fail()
}

func (u *unavailableConnector) FetchDailySales(ctx context.Context, startDate, endDate time.Time) ([]model.DailySalesMetric, error) {
	return nil, u.fail()
}

func (u *unavailableConnector) fail() error {
	return fmt.Errorf("connector %s unavailable in production: %w", u.code, u.reason)
}

// BuildConnectors wires the per-channel connector list (#62): each platform
// uses its real API connector when the required credentials are present in
// the environment. Without credentials, development falls back to the
// deterministic mock connector (dashboard keeps working out of the box),
// while production installs an unavailable connector that writes nothing —
// fabricated numbers must never reach a production database.
func BuildConnectors(getenv EnvSource, strict bool) []PlatformConnector {
	if getenv == nil {
		getenv = osGetenv
	}

	builders := []struct {
		channel string
		name    string
		build   func(EnvSource) (PlatformConnector, error)
		mock    func() PlatformConnector
	}{
		{"meta_ads", "Meta Ads", buildMetaFromEnv, NewMetaConnector},
		{"tiktok_shop", "TikTok Shop", buildTikTokFromEnv, NewTikTokConnector},
		{"shopee", "Shopee", buildShopeeFromEnv, NewShopeeConnector},
	}

	connectors := make([]PlatformConnector, 0, len(builders))
	for _, b := range builders {
		conn, err := b.build(getenv)
		switch {
		case err == nil:
			logConnectorMode(b.channel, "real-api", "")
			connectors = append(connectors, conn)
		case strict:
			logConnectorMode(b.channel, "unavailable", err.Error())
			connectors = append(connectors, &unavailableConnector{code: b.channel, name: b.name, reason: err})
		default:
			logConnectorMode(b.channel, "mock", err.Error())
			connectors = append(connectors, b.mock())
		}
	}
	return connectors
}

func buildMetaFromEnv(getenv EnvSource) (PlatformConnector, error) {
	return NewMetaAPIConnector(
		getenv("META_ACCESS_TOKEN"),
		getenv("META_AD_ACCOUNT_ID"),
		getenv("META_API_BASE"),
		getenv("META_API_VERSION"),
	)
}

func buildTikTokFromEnv(getenv EnvSource) (PlatformConnector, error) {
	return NewTikTokAPIConnector(
		getenv("TIKTOK_ACCESS_TOKEN"),
		getenv("TIKTOK_ADVERTISER_ID"),
		getenv("TIKTOK_API_BASE"),
	)
}

func buildShopeeFromEnv(getenv EnvSource) (PlatformConnector, error) {
	return NewShopeeAPIConnector(
		getenv("SHOPEE_PARTNER_ID"),
		getenv("SHOPEE_PARTNER_KEY"),
		getenv("SHOPEE_ACCESS_TOKEN"),
		getenv("SHOPEE_SHOP_ID"),
		getenv("SHOPEE_API_BASE"),
		getenv("SHOPEE_CAMPAIGN_IDS"),
		getenv("SHOPEE_TIMEZONE"),
	)
}
