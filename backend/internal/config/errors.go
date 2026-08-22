package config

import "errors"

// ErrMissingAPIKey is returned by Config.Validate when the server runs in
// production without an API key, which would leave every endpoint (including
// POST /api/v1/sync) unauthenticated.
var ErrMissingAPIKey = errors.New("APP_ENV=production requires API_KEY to be set; refusing to start with authentication disabled")
