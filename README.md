# 🗓️ Clandar

A self-hosted global public holiday calendar

## Overview

Clandar is a web app that shows public holidays for 20 countries across ASEAN, Asia-Pacific, Europe, and the Americas. It runs as a single self-contained binary backed by SQLite — no external database required.

Key features:
- Filterable by region, country, and holiday type (public, optional, observance)
- Year navigation (past and future years)
- Holiday detail popovers with name, date, and type
- Dark UI

> 📸 _Screenshot coming soon_

## Countries covered

| Region | Countries |
|--------|-----------|
| ASEAN | Brunei, Cambodia, Indonesia, Laos, Malaysia, Myanmar, Philippines, Singapore, Thailand, Vietnam |
| Asia-Pacific | Australia, China, Japan |
| Europe | France, Germany, Italy, Netherlands, Spain, United Kingdom |
| Americas | United States |

## Data sources

- **[Nager.Date](https://date.nager.at)** — free, no API key needed. Covers public holidays for most supported countries.
- **[Calendarific](https://calendarific.com)** — optional, free tier available. Provides broader coverage including religious and cultural holidays, and fills in countries that Nager.Date does not support.

## Getting started

### Option A — Run locally

```bash
git clone https://github.com/barayuda/clandar
cd clandar/Clandar
cp .env.example .env        # edit CALENDARIFIC_API_KEY if you have one
go mod tidy
go run ./cmd/server
# Open http://localhost:8080
```

### Option B — Docker (recommended)

```bash
git clone https://github.com/barayuda/clandar
cd clandar/Clandar
cp .env.example .env        # edit CALENDARIFIC_API_KEY if you have one
docker compose up --build
# Open http://localhost:8080
```

> **Note:** On first boot the server automatically seeds all 20 countries and fetches holiday data. This takes approximately 10–15 seconds before holidays appear in the UI.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Logging level (`debug` / `info` / `warn` / `error`) |
| `DB_PATH` | `./data/clandar.db` | Path to the SQLite database file |
| `CALENDARIFIC_API_KEY` | _(empty)_ | Optional — improves coverage for ASEAN countries |

Copy `.env.example` to `.env` and set values there. The `.env` file is never committed.

## API

| Endpoint | Query params | Description |
|----------|-------------|-------------|
| `GET /api/holidays` | `country`, `region`, `year`, `type` | Returns holidays matching the given filters |
| `GET /api/countries` | `region` | Returns all countries, optionally filtered by region |
| `GET /api/regions` | — | Returns the 4 regions with country counts |
| `GET /health` | — | Health check |

Example requests:

```bash
curl "http://localhost:8080/api/holidays?country=SG&year=2026"
curl "http://localhost:8080/api/holidays?region=ASEAN&year=2026&type=public"
```

## Testing

```bash
# Go unit and integration tests
go test ./internal/...

# Playwright E2E tests (requires the server to be running on :8080)
cd tests/e2e
npm install && npx playwright install chromium
npm test
```

## Project structure

```
Clandar/
├── cmd/server/        # main entry point
├── internal/
│   ├── api/           # HTTP handlers & router
│   ├── fetcher/       # Nager.Date & Calendarific clients
│   ├── scheduler/     # yearly auto-sync
│   ├── seeder/        # country seeding & holiday sync
│   └── store/         # SQLite queries & models
├── db/                # schema.sql & sqlc queries
├── web/               # frontend (single HTML file)
├── tests/e2e/         # Playwright specs
├── Dockerfile
└── docker-compose.yml
```

## Built with

- [Go 1.22](https://go.dev), [chi](https://github.com/go-chi/chi) router, [zerolog](https://github.com/rs/zerolog), [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) (pure Go — no CGo required)
- [Playwright](https://playwright.dev) for E2E testing
- Data: [Nager.Date API](https://date.nager.at), [Calendarific API](https://calendarific.com)
