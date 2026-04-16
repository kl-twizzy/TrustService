# Seller Trust Map

Diploma project: a service for seller reputation analysis and product rating authenticity checks on marketplaces.

## Stack

- Go backend API
- Python FastAPI anti-fraud service
- PostgreSQL
- Redis
- Browser extension
- Docker Compose

## Project structure

- `backend-go` - main API, URL parsing, trust score calculation, PostgreSQL persistence
- `ml-service` - anomaly and suspicious review analysis
- `browser-extension` - popup UI for checking the current product page
- `infra/postgres` - schema and seed data
- `docs` - architecture and diploma notes
- `demo` - local demo page

## What is implemented

- trust score calculation for seller and product
- support for real product URLs from `OZON`, `Wildberries`, `Yandex Market`
- fallback analysis for unknown product IDs from real links
- PostgreSQL schema for marketplaces, sellers, products, reviews, metrics, runs, snapshots
- storing completed checks in PostgreSQL
- recent checks endpoint: `GET /api/v1/checks/recent`
- per-user history using `client_id`
- backend-side page enrichment from real marketplace HTML
- Redis cache with TTL 1 hour
- browser extension popup with `Check product` button

## Quick start with Docker

```bash
docker compose up --build
```

After startup:

- backend API: `http://localhost:8080`
- ML service: `http://localhost:8001`
- health check: `http://localhost:8080/health`
- local dashboard: `http://localhost:8080/dashboard`

## Using your local PostgreSQL

If PostgreSQL is already installed on your PC:

1. Create database `seller_trust`
2. Run [infra/postgres/init.sql](C:/Users/kiril/OneDrive/Документы/diplom/infra/postgres/init.sql) in pgAdmin4
3. Set environment variable before starting backend:

```powershell
$env:DATABASE_URL="postgres://postgres:postgres@localhost:5432/seller_trust?sslmode=disable"
```

In this mode backend will:

- read sellers, products and reviews from PostgreSQL
- save analysis runs and trust snapshots
- return recent checks history

If `DATABASE_URL` is not set or PostgreSQL is unavailable, backend falls back to generated demo data.

## Browser extension flow

1. Open `chrome://extensions/`
2. Enable developer mode
3. Click `Load unpacked`
4. Select [browser-extension](C:/Users/kiril/OneDrive/Документы/diplom/browser-extension)
5. Open a product page on `OZON`, `Wildberries` or `Yandex Market`
6. Click the extension icon
7. Click `Проверить товар`

The popup automatically takes the URL of the active tab and sends it to backend.

## Local defense demo

For a simple local presentation:

1. Start the project with `docker compose up --build`
2. Open `http://localhost:8080/dashboard`
3. Paste a real marketplace product URL
4. Run analysis
5. Show:
   - calculated trust score
   - rating authenticity
   - risk reasons
   - your personal checks history
   - page signals extracted by backend from HTML
   - persisted backend flow

## API examples

Check by product URL:

```bash
curl -X POST http://localhost:8080/api/v1/trust/analyze-url \
  -H "Content-Type: application/json" \
  -d "{\"product_url\":\"https://www.ozon.ru/product/test-2530782332/\"}"
```

Recent checks:

```bash
curl http://localhost:8080/api/v1/checks/recent
```

Overview:

```bash
curl http://localhost:8080/api/v1/overview
```

## Main scenario

1. User opens a marketplace product page
2. Browser extension gets the active tab URL
3. Go backend detects marketplace and extracts product ID from URL
4. Backend tries PostgreSQL first
5. If product is missing in DB, backend builds fallback product/seller data from URL/product ID
6. Backend calls Python anti-fraud service
7. Backend calculates final trust score
8. Result is cached in Redis and stored in PostgreSQL history tables
9. Checks can be filtered per local user/browser session via `client_id`
9. Extension shows trust score, authenticity score and risk reasons

## Current status

This is now a solid MVP for a diploma project:

- real URL-based check flow exists
- data persistence exists
- anti-fraud service exists
- browser extension exists

The next major step would be real marketplace page parsing and data collection instead of fallback/generated product data.
