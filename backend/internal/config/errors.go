package config

import "errors"

// ErrMissingAPIKey is returned by Config.Validate when the server runs in
// production without an API key, which would leave every endpoint (including
// POST /api/v1/sync) unauthenticated.
var ErrMissingAPIKey = errors.New("APP_ENV=production requires API_KEY to be set; refusing to start with authentication disabled")

// ErrInvalidEnv is returned by Config.Validate when APP_ENV holds an unknown
// profile: guessing between strict and lax behavior is unsafe, so only
// development/dev and production/prod are accepted.
var ErrInvalidEnv = errors.New("APP_ENV must be one of: development, dev, production, prod")
