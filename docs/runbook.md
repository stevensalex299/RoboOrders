# RoboOrders — runbook

How to clone the repo, run the Go API, inject webhook sample orders, and (optionally) run the Vue UI.

Architecture and design: [decisionNotes.md](decisionNotes.md).

## Prerequisites

- **Go 1.22+** ([install](https://go.dev/dl/)) — Go module lives in `backend/`
- **Node.js 18+** (only if running the frontend; Vite 6)
- **curl** (optional, for manual API checks)

## Repository layout

| Path | Purpose |
|------|---------|
| `backend/` | Go module: API server (`cmd/server`) and ingest CLI (`cmd/ingest`) |
| `frontend/` | Vue 3 + Tailwind dev UI |
| `fixtures/` | Sample webhook jsonl, API jsonl, CSV files |

---

## Quick start (reviewer path)

Use **two terminals**. All `go run` commands below assume your working directory is **`backend/`** (required for the Go module).

### 1. Start the API

```bash
cd backend
go run ./cmd/server
```

You should see: `RoboOrders API listening on :8080`.

Leave this running. If port 8080 is already in use, stop the other process or set `ROBO_ORDERS_ADDR=:8081` and use that port in curl/ingest `--http` (the Vue dev proxy expects **8080** by default).

**First run:** Go will download modules automatically. If needed: `go mod download`.

### 2. Verify the API

In a second terminal:

```bash
curl -s http://localhost:8080/health
```

Expected: `{"status":"ok"}`

### 3. Load sample webhook orders (recommended)

Still from **`backend/`**:

```bash
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 20 -q
```

This replays the **first 20 lines** of `fixtures/webhook_orders.jsonl` into SQLite (**sequential**, one order per line). The server does not need to be stopped; ingest uses the same database file as the API.

Confirm orders exist:

```bash
curl -s http://localhost:8080/orders | head -c 200
```

### 4. (Optional) Start the UI

From repo root:

```bash
cd frontend
npm install
npm run dev
```

Open the URL printed in the terminal (usually **http://localhost:5173**). The UI polls `GET /orders` through the Vite proxy to **http://localhost:8080**.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ROBO_ORDERS_ADDR` | `:8080` | API listen address |
| `ROBO_ORDERS_DB` | `data/roboorders.db` (relative to **`backend/`** cwd) | SQLite path |

Schema is applied automatically on startup (`CREATE TABLE IF NOT EXISTS`). The DB file is gitignored under `backend/data/`.

To reset local data: stop the server, delete `backend/data/roboorders.db`, restart the server.

---

## HTTP API (webhook + reads)

Base URL: **http://localhost:8080** (unless you changed `ROBO_ORDERS_ADDR`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/orders` | List orders; optional `?status=received\|scheduled\|dispatched\|cancelled` |
| GET | `/orders/{id}` | One order by hub UUID (404 if missing) |
| POST | `/webhooks/orders` | Ingest one webhook JSON body (same shape as one line of `webhook_orders.jsonl`) |

### Examples (from any directory)

List all orders:

```bash
curl -s 'http://localhost:8080/orders'
```

Filter by status:

```bash
curl -s 'http://localhost:8080/orders?status=received'
```

Single webhook (manual; run from repo root so the path resolves):

```bash
head -1 fixtures/webhook_orders.jsonl | curl -s -X POST http://localhost:8080/webhooks/orders \
  -H 'Content-Type: application/json' -d @-
```

Cancel an existing webhook order: POST again with the same `order_id` and `"update": ["cancelled"]` in the JSON (see cancel lines in the fixture file), or replay a cancel line via the ingest CLI.

---

## Ingest CLI — webhook mock

**Important:** Run from **`backend/`** (where `go.mod` lives):

```bash
cd backend
go run ./cmd/ingest help
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl [options]
```

### Modes

| Mode | Flags | Behavior |
|------|-------|----------|
| **Direct DB** (default) | no `--http` | Opens `ROBO_ORDERS_DB` and upserts each line. Server may be running (same DB file). |
| **Via API** | `--http http://localhost:8080` | POST each line to `/webhooks/orders`. Server **must** be running. |

Lines are always processed **sequentially** (one after another).

### Common options

| Flag | Meaning |
|------|---------|
| `--file PATH` | **Required.** Path to `webhook_orders.jsonl` (from `backend/`, use `../fixtures/webhook_orders.jsonl`) |
| `--from N` | First line number, 1-based (default `1`) |
| `--to N` | Last line inclusive (`0` = end of file) |
| `--limit N` | Max lines to ingest after `--from` |
| `--delay MS` | Pause between lines (milliseconds) |
| `-q` | Print summary only |

### Examples (from `backend/`)

```bash
# First 5 orders
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 5

# Lines 10–20 inclusive
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --from 10 --to 20

# Replay 50 orders through the HTTP API
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --http http://localhost:8080 --limit 50 -q

# One cancel payload from the fixture (line 64)
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --from 64 --limit 1
```

Successful runs print a line like: `webhook ingest: N/N ok in ... mode=db` or `mode=http`.

---

## Inject sample orders (summary)

| Source | Status | How |
|--------|--------|-----|
| **Webhook** | Implemented | `go run ./cmd/ingest webhook` or `POST /webhooks/orders` |
| **Poll** | Planned | CLI over `fixtures/api_responses.jsonl` |
| **CSV** | Planned | CLI over `fixtures/orders_*.csv` |

---

## Fixtures

Sample data lives in [`fixtures/`](../fixtures/) at the repo root:

- `webhook_orders.jsonl` — one JSON order per line (`order_id`, names, `items`, optional `"update": ["cancelled"]`)
- `api_responses.jsonl` — poll mock (not wired yet)
- `orders_1.csv` … `orders_4.csv` — CSV mock (not wired yet)

---

## Troubleshooting

| Issue | What to do |
|-------|------------|
| `go: cannot find main module` | Run `go run` from **`backend/`**, not repo root |
| `address already in use` on 8080 | Stop the old process (`lsof -i :8080`) or change `ROBO_ORDERS_ADDR` |
| UI empty but curl works | Ensure API is on **8080** (Vite proxy default) |
| Ingest ok but UI unchanged | Wait up to 5s for poll, or change status filter; confirm same DB (default `backend/data/roboorders.db`) |

---

## Not implemented yet

- `GET /orders/{id}/events`, dispatch, poll ingest, CSV ingest

See [decisionNotes.md](decisionNotes.md) for planned behavior.
