#!/usr/bin/env bash
# =============================================================================
# scripts/commit.sh — Clandar semantic commit history builder
#
# Usage:
#   bash scripts/commit.sh
#
# Requirements:
#   - GPG key configured for git signing (git config user.signingkey)
#   - Run from the project root: /path/to/clandar/Clandar/
#
# What it does:
#   1. Initialises a git repo (if not already done)
#   2. Stages files and creates each signed, semantic commit in order
#   3. Stops immediately on any error (set -e)
#
# To verify signed commits after running:
#   git log --show-signature --oneline
# =============================================================================

set -euo pipefail

# ── Helpers ──────────────────────────────────────────────────────────────────

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
RESET='\033[0m'

log() { echo -e "${CYAN}▶ $*${RESET}"; }
ok() { echo -e "${GREEN}✓ $*${RESET}"; }
die() {
  echo -e "${RED}✗ $*${RESET}"
  exit 1
}

commit() {
  local message="$1"
  shift
  # "$@" contains the files to stage
  git add -- "$@"
  git commit -S -s -m "$message"
  ok "Committed: $message"
}

# ── Pre-flight checks ─────────────────────────────────────────────────────────

# Must be run from project root (where go.mod lives)
[[ -f go.mod ]] || die "Run this script from the Clandar project root (go.mod not found)"

# Git must be available
command -v git &>/dev/null || die "git not found in PATH"

# GPG signing key must be configured
SIGNING_KEY=$(git config --global user.signingkey 2>/dev/null || git config user.signingkey 2>/dev/null || true)
[[ -n "$SIGNING_KEY" ]] || die "No GPG signing key configured.\nRun: git config --global user.signingkey <YOUR_KEY_ID>"

log "GPG signing key: $SIGNING_KEY"

# ── Init repo if needed ───────────────────────────────────────────────────────

if [[ ! -d .git ]]; then
  log "Initialising git repository..."
  git init
  ok "git init done"
else
  log "Git repo already initialised — continuing"
fi

# Ensure commit signing is enabled for this repo
git config commit.gpgsign true

# ── Commits ───────────────────────────────────────────────────────────────────

log "Starting commit sequence (19 commits)..."
echo ""

# 1 ─ Project scaffold
commit \
  "chore: initialize Go module and project scaffold" \
  go.mod \
  go.sum \
  .gitignore \
  .env.example

# 2 ─ Config
commit \
  "feat(config): add environment-based configuration loader" \
  internal/config/config.go

# 3 ─ Store layer
commit \
  "feat(store): add SQLite schema, query definitions, and store layer" \
  db/schema.sql \
  db/queries.sql \
  db/sqlc.yaml \
  internal/store/db.go \
  internal/store/models.go \
  internal/store/methods.go

# 4 ─ Fetchers
commit \
  "feat(fetcher): add Nager.Date and Calendarific holiday API clients" \
  internal/fetcher/nager.go \
  internal/fetcher/calendarific.go \
  internal/fetcher/fetcher.go

# 5 ─ Seeder + scheduler
commit \
  "$(printf 'feat(seeder): add country seeder and annual sync scheduler\n\nSeeds 20 priority countries across ASEAN, Asia-Pacific, Europe,\nand Americas on first boot. Scheduler re-fetches new year data\nevery January 1 UTC.')" \
  internal/seeder/seeder.go \
  internal/scheduler/scheduler.go

# 6 ─ API
commit \
  "$(printf 'feat(api): add chi router with holidays, countries, and regions endpoints\n\nSupports filtering by country, region, year (2000-2100), and\nholiday type. Returns JSON with Cache-Control headers.')" \
  internal/api/router.go \
  internal/api/handler.go

# 7 ─ Server entrypoint
commit \
  "feat(server): add main entry point with graceful shutdown" \
  cmd/server/main.go

# 8 ─ Docker
commit \
  "$(printf 'build(docker): add multi-stage Dockerfile and docker-compose\n\nTwo-stage build: golang:1.22-alpine builder to alpine:3.20 runtime.\nPure-Go SQLite driver (modernc.org/sqlite) — no CGo required.\nIncludes wget for health checks and non-root user for security.')" \
  Dockerfile \
  docker-compose.yml

# 9 ─ Frontend
commit \
  "$(printf 'feat(frontend): add single-file calendar UI\n\nDark-theme SPA with 12-month grid, region/country filter sidebar,\nholiday type chips, year navigation, and holiday detail popovers.\nFetches live data from the Go API; caches per year+filter.')" \
  web/index.html

# 10 ─ Store tests
commit \
  "test(store): add SQLite store integration tests" \
  internal/store/store_test.go

# 11 ─ API tests
commit \
  "test(api): add HTTP handler unit tests with in-memory SQLite" \
  internal/api/handler_test.go

# 12 ─ Fetcher tests
commit \
  "$(printf 'test(fetcher): add Nager.Date fetcher unit tests with httptest mock server\n\nCovers 200 success, 204 no-content, 404 not-found, 500 server error,\nand type mapping table. Zero real network calls.')" \
  internal/fetcher/nager_test.go

# 13 ─ E2E tests
commit \
  "$(printf 'test(e2e): add Playwright E2E test suite\n\n14 specs across calendar UI and API layers. Browser tests cover\nyear navigation, holiday popovers, region filtering, Escape to close,\nand zero console errors. API tests cover all endpoints and error cases.')" \
  tests/e2e/.gitkeep \
  tests/e2e/package.json \
  tests/e2e/package-lock.json \
  tests/e2e/playwright.config.ts \
  tests/e2e/specs/api.spec.ts \
  tests/e2e/specs/calendar.spec.ts

# 14 ─ Docs
commit \
  "docs: add README with setup guide, API reference, and project structure" \
  README.md

# 15 ─ UI improvements
commit \
  "$(printf 'feat(ui): add glanceability improvements to calendar UI\n\n- Next holiday pill in sidebar (auto-scrolls to that month on click)\n- Jump-to-today button in year navigation\n- Days-away counter in holiday detail popover\n- Multi-dot density per day (up to 3, colour-coded by type)\n- Heatmap tint on month headers based on holiday count\n- Weekend column shading (SAT/SUN)\n- Current month card highlighted with accent border\n- Country search box in sidebar filter\n- Holiday count badge per country\n- Clear filters button in sidebar footer')" \
  web/index.html

# 16 ─ ASEAN expansion
commit \
  "$(printf 'feat(seeder): add Indonesia and full ASEAN country coverage\n\nAdds 7 additional ASEAN countries: Brunei, Cambodia, Laos,\nMalaysia, Myanmar, Philippines, Thailand, Vietnam.\nMoves ASEAN countries to their own region group.\nTotal: 20 countries across 4 regions.')" \
  internal/seeder/seeder.go

# 17 ─ Render deployment
commit \
  "$(printf 'build(render): add render.yaml Blueprint and harden Docker for Render\n\nrender.yaml provisions a Web Service (Singapore region, free plan).\nNo persistent disk — database handled by Turso cloud SQLite.\nDockerfile pinned to alpine:3.20, non-root user, HEALTHCHECK added.\ndocker-compose.yml uses \${PORT:-8080} for env-driven port mapping.')" \
  render.yaml \
  Dockerfile \
  docker-compose.yml

# 18 ─ Turso cloud SQLite integration
commit \
  "$(printf 'feat(store): add Turso cloud SQLite support with local fallback\n\nAdds libsql-client-go driver alongside modernc.org/sqlite.\nstore.Open() selects driver based on TURSO_DATABASE_URL env var:\n  - set → connects to Turso cloud (production / Render free)\n  - unset → opens local SQLite file (local dev unchanged)\nconfig.go gains TursoDatabaseURL, TursoAuthToken, IsRemoteDB().\nrender.yaml updated to free plan with no persistent disk.\n.env.example documents new Turso env vars.')" \
  internal/config/config.go \
  internal/store/db.go \
  cmd/server/main.go \
  render.yaml \
  .env.example \
  go.mod \
  go.sum

# 19 ─ Commit helper script
commit \
  "chore(scripts): add semantic commit helper script" \
  scripts/commit.sh

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
ok "All 19 commits created successfully."
echo ""
log "Summary:"
git log --oneline
echo ""
log "To verify signatures:"
echo "  git log --show-signature --oneline"
echo ""
log "To push to GitHub:"
echo "  git remote add origin git@github.com:barayuda/clandar.git"
echo "  git push -u origin main"
