# E-Commerce & Marketing Analytics Engine

Full-stack marketing analytics and attribution engine for aggregating, normalizing, and visualizing advertising performance and e-commerce sales across multiple digital marketing channels (Meta Ads, TikTok Shop, and Shopee).

## 🏗 Architecture & Tech Stack

- **Backend:** Go 1.22+ (Modular Monolith with standard net/http, modernc.org/sqlite, sync services, background schedulers)
- **Frontend:** Next.js 14 (App Router), React 18, TypeScript, Tailwind CSS, Recharts, Lucide Icons
- **Database:** SQLite with idempotent upsert repository pattern

## 🚀 Quick Start

### 1. Environment Configuration

Before running the application, copy the example environment configuration files:

**Backend:**
```bash
cp backend/.env.example backend/.env
```

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP port for the backend server |
| `DB_PATH` | `analytics.db` | Path to the SQLite database file |
| `SYNC_INTERVAL_MINUTES` | `60` | Background ingestion interval in minutes |

**Frontend:**
```bash
cp frontend/.env.example frontend/.env.local
```

| Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://localhost:8080/api/v1` | Base URL for the backend API |

---

### 2. Backend Setup & Run

```bash
cd backend
go mod download
go run cmd/server/main.go
```

The REST API server will run on http://localhost:8080.

To run backend tests:
```bash
cd backend
go test ./...
```

---

### 3. Frontend Setup & Run

```bash
cd frontend
npm install
npm run dev
```

The Next.js dashboard will be available at http://localhost:3000.

To build frontend for production:
```bash
cd frontend
npm run build
```

## 📊 Features & KPI Engine

- **Aggregated Performance Metrics:** Total Spend, Total GMV, Blended ROAS, Channel ROAS, Average CPA, ACOS, and Net Contribution Margin.
- **Multi-Channel Data Connectors:** Modular connector interface with mock adapters for Meta Ads, TikTok Shop, and Shopee.
- **Concurrent Ingestion Pipeline:** Automated background synchronization scheduler and on-demand sync triggers.
- **Interactive Analytics Dashboard:** Real-time KPI summary cards, dual-axis performance trend charts, channel distribution breakdown, and searchable campaign table with pagination.

## 📄 Documentation

- [Design Specification](docs/superpowers/specs/2026-08-21-ecommerce-analytics-engine-design.md)
- [Implementation Plan](docs/superpowers/plans/2026-08-21-ecommerce-analytics-engine.md)
