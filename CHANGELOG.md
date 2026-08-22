# Changelog

All notable changes to this project will be documented in this file.

## [1.0.0] - 2026-08-21

### Added
- **Security:**
  - Strict CSRF protection with custom headers on state-mutating sync endpoints.
  - Granular API key authentication with Bearer token fallback.
  - In-memory token bucket rate limiting with HTTP 429 throttling.
  - Strict input bounds validation and pagination query limits.
- **Backend Architecture & Performance:**
  - SQL aggregation optimizations replacing in-memory dataset iteration.
  - Composite and lookup database indexes for campaigns, metrics, and sync logs.
  - Composable HTTP middleware chain (`Recovery`, `Logging`, `CORS`, `RateLimit`, `Auth`).
  - Cache-Control response headers for analytics endpoints.
  - Bounded concurrency semaphore for multi-channel synchronization.
  - Go 1.22+ `math/rand/v2` with deterministic PCG seeding.
- **Frontend Architecture & UX:**
  - Next.js 14 App Router dashboard with responsive Tailwind UI.
  - Main dashboard loading state skeletons and App Router `loading.tsx`.
  - Campaign search 300ms debounce handler to eliminate redundant requests.
  - Resilient API client with configurable timeouts (`AbortController`) and exponential backoff retry.
  - Full test coverage with Vitest for client utilities and components.
- **DevOps & CI/CD:**
  - Multi-stage Alpine containerization for backend and frontend with non-root security.
  - GitHub Actions CI workflows with immutable SHA action pinning and vulnerability scanning (`govulncheck`).
  - Complete OpenAPI 3.0 API specification and documentation.
