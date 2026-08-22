package connector

import (
	"os"
)

// EnvSource abstracts environment lookup so tests can drive the factory
// without mutating the process environment.
type EnvSource func(string) string

func osGetenv(key string) string { return os.Getenv(key) }

// BuildConnectors wires the per-channel connector list (#62): each platform
// uses its real API connector when the required credentials are present in
// the environment, and falls back to the deterministic mock connector (with
// a warning) otherwise, so the dashboard keeps working out of the box.
func BuildConnectors(getenv EnvSource) []PlatformConnector {
	if getenv == nil {
		getenv = osGetenv
	}

	builders := []struct {
		channel string
		build   func(EnvSource) (PlatformConnector, error)
		mock    func() PlatformConnector
	}{
		{"meta_ads", buildMetaFromEnv, NewMetaConnector},
		{"tiktok_shop", buildTikTokFromEnv, NewTikTokConnector},
		{"shopee", buildShopeeFromEnv, NewShopeeConnector},
	}

	connectors := make([]PlatformConnector, 0, len(builders))
	for _, b := range builders {
		conn, err := b.build(getenv)
		if err == nil {
			logConnectorMode(b.channel, "real-api", "")
			connectors = append(connectors, conn)
			continue
		}
		logConnectorMode(b.channel, "mock", err.Error())
		connectors = append(connectors, b.mock())
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
	)
}
