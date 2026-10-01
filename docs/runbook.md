# RoboOrders — runbook

How to run the API, load sample orders, use the web UI, and call the HTTP API. Architecture and design: [decisionNotes.md](decisionNotes.md).

## Prerequisites

- **Go 1.22+** ([install](https://go.dev/dl/)) — Go module lives in `backend/`
- **Node.js 18+** (for the frontend; Vite 6)
- **curl** (optional; examples below use it)

## Repository layout

| Path | Purpose |
|------|---------|
| `backend/` | Go module: API server (`cmd/server`) and ingest CLI (`cmd/ingest`) |
| `frontend/` | Vue 3 + Tailwind UI |
| `fixtures/` | Sample webhook jsonl, API jsonl, CSV files |

---

## Quick start

Use **two terminals**. All `go run` commands assume your working directory is **`backend/`** (required for the Go module).

### 1. Start the API

```bash
cd backend
go run ./cmd/server
```

The server listens on **:8080** by default (`RoboOrders API listening on :8080`). If port 8080 is in use, stop the other process or set `ROBO_ORDERS_ADDR=:8081` and use that port for curl/ingest `--http` (the Vue dev proxy expects **8080** by default).

On first run, Go downloads modules automatically (`go mod download` if needed).

### 2. Load sample webhook orders

From **`backend/`** (API may keep running):

```bash
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 20 -q
```

This replays the first 20 lines of `fixtures/webhook_orders.jsonl` into SQLite, one order per line, sequentially. Ingest uses the same database file as the running API.

### 3. Start the UI (optional)

```bash
cd frontend
npm install
npm run dev
```

Open the URL Vite prints (usually **http://localhost:5173**). The UI talks to the API through the dev proxy on **http://localhost:8080**.

---

## Using the web UI

- **Order list** — All orders refresh every 5 seconds. Use the status dropdown to filter (`received`, `scheduled`, `dispatched`, `cancelled`).
- **Order detail** — Click a customer name to open that order: metadata, line items, and event history.
- **Dispatch** — On the detail page, **Dispatch to robot** is available when the order is `received` or `scheduled`. It sets status to `dispatched` and appends an `order_dispatched` event (robot payload is stored on the event).

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `ROBO_ORDERS_ADDR` | `:8080` | API listen address |
| `ROBO_ORDERS_DB` | `data/roboorders.db` (relative to **`backend/`** cwd) | SQLite path |

Schema is applied on startup. The DB file is gitignored under `backend/data/`.

To reset local data: stop the server, delete `backend/data/roboorders.db`, restart the server.

---

## HTTP API

Base URL: **http://localhost:8080** (unless you changed `ROBO_ORDERS_ADDR`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/orders` | List orders; optional `?status=received\|scheduled\|dispatched\|cancelled` |
| GET | `/orders/{id}` | One order by hub UUID |
| GET | `/orders/{id}/events` | Event history for an order |
| POST | `/orders/{id}/dispatch` | Manual dispatch (`received` or `scheduled` only) |
| POST | `/webhooks/orders` | Ingest one webhook JSON body (same shape as one line of `webhook_orders.jsonl`) |

### Examples

List orders:

```bash
curl -s 'http://localhost:8080/orders'
curl -s 'http://localhost:8080/orders?status=received'
```

Ingest one webhook line from the fixture (from repo root):

```bash
head -1 fixtures/webhook_orders.jsonl | curl -s -X POST http://localhost:8080/webhooks/orders \
  -H 'Content-Type: application/json' -d @-
```

Cancel: POST again with the same `order_id` and `"update": ["cancelled"]`, or replay a cancel line via the ingest CLI (e.g. fixture line 64).

Order detail, events, and dispatch (replace `{id}` with a hub UUID from `GET /orders`):

```bash
curl -s "http://localhost:8080/orders/{id}"
curl -s "http://localhost:8080/orders/{id}/events"
curl -s -X POST "http://localhost:8080/orders/{id}/dispatch"
```

---

## Ingest CLI — webhook

Run from **`backend/`**:

```bash
go run ./cmd/ingest help
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl [options]
```

### Modes

| Mode | Flags | Behavior |
|------|-------|----------|
| **Direct DB** (default) | no `--http` | Upserts into `ROBO_ORDERS_DB`. API may be running (same file). |
| **Via API** | `--http http://localhost:8080` | POST each line to `/webhooks/orders`. API **must** be running. |

Lines are processed **sequentially**.

### Options

| Flag | Meaning |
|------|---------|
| `--file PATH` | **Required.** e.g. `../fixtures/webhook_orders.jsonl` |
| `--from N` | First line, 1-based (default `1`) |
| `--to N` | Last line inclusive (`0` = end of file) |
| `--limit N` | Max lines after `--from` |
| `--delay MS` | Pause between lines |
| `-q` | Summary output only |

### Examples

```bash
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --limit 5
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --from 10 --to 20
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --http http://localhost:8080 --limit 50 -q
go run ./cmd/ingest webhook --file ../fixtures/webhook_orders.jsonl --from 64 --limit 1
```

---

## Data sources (fixtures)

| Source | How to load today |
|--------|-------------------|
| **Webhook** | Ingest CLI or `POST /webhooks/orders` |
| **Poll** | Not implemented — fixture: `api_responses.jsonl` |
| **CSV** | Not implemented — fixtures: `orders_1.csv` … `orders_4.csv` |

Files live in [`fixtures/`](../fixtures/):

- `webhook_orders.jsonl` — one JSON order per line (`order_id`, names, `items`, optional `"update": ["cancelled"]`)
- `api_responses.jsonl` — poll mock (future)
- `orders_*.csv` — CSV mock (future)

Further behavior: [decisionNotes.md](decisionNotes.md).

---

## Troubleshooting

| Issue | What to do |
|-------|------------|
| `go: cannot find main module` | Run `go run` from **`backend/`**, not repo root |
| `address already in use` on 8080 | Stop the old process (`lsof -i :8080`) or change `ROBO_ORDERS_ADDR` |
| UI shows no orders | Ingest sample data; ensure API is on **8080** (Vite proxy default) |
| Ingest ran but UI unchanged | Wait up to 5s for refresh, or check status filter; same DB as API (`backend/data/roboorders.db` by default) |
