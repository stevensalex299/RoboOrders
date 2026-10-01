# RoboOrders — runbook

How to run the API, load sample orders, use the web UI, and call the HTTP API. Architecture and design: [decisionNotes.md](decisionNotes.md).

## Prerequisites

- **Go 1.22+** ([install](https://go.dev/dl/)) — Go module lives in `backend/`
- **Node.js 18+** (for the frontend; Vite 6)
- **curl** (optional; examples below use it)

Shell examples below assume macOS/Linux or **Git Bash** on Windows; **`go run ./cmd/ingest …`** and **`npm run dev`** work the same from **`backend/`** and **`frontend/`** in PowerShell—only `curl`/`head` one-liners may need Git Bash, WSL, or PowerShell equivalents.

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

Open the URL Vite prints (usually **http://localhost:5173**). The UI talks to the API through the dev proxy on **http://localhost:8080**. Hard refresh on `/orders/{id}` works in dev (SPA fallback through the proxy).

### 4. Load poll and CSV (optional, same DB)

Still from **`backend/`**:

```bash
# Poll: one fixture line per run by default; repeat to advance the cursor
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl --limit 5 -q
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl --limit 5 -q

# CSV: one or more survey files (repeat --file)
go run ./cmd/ingest csv --file ../fixtures/orders_1.csv --file ../fixtures/orders_2.csv -q
```

Reload the UI (or wait up to 5s). You should see orders from all three sources.

---

## Typical workflow (reviewer path)

1. **Start API** (`go run ./cmd/server` from `backend/`).
2. **Start UI** (`npm run dev` from `frontend/`), open **http://localhost:5173**.
3. **Ingest fixtures** from `backend/` — webhook first, then poll in several small batches, then CSV (see [Ingest CLI](#ingest-cli--webhook) sections).
4. **Browse the list** — use **Source** and **Status** dropdowns; use **Previous / Next** at the bottom (10 orders per page). The list auto-refreshes every **5 seconds**.
5. **Open an order** — click the customer name (poll orders without names show as **Order {number}**).
6. **Inspect** — line items (poll lines include statuses like `ordered` / `processing`), **Scheduled for** on CSV when applicable, and **Event history**.
7. **Dispatch** — on a `received` or `scheduled` order, click **Dispatch to robot**; status becomes `dispatched` and a new event appears.
8. **Go back** — **← Back to orders** restores your list **source**, **status**, and **page** (stored in the URL query string).

To explore **cancel** and **line status changes**, ingest webhook line **64** (Jamie cancel) and run **poll** until batches include status updates (see poll section).

---

## Using the web UI

### Order list

- **Filters** — **Source**: All, Webhook, External poll, CSV. **Status**: All, received, scheduled, dispatched, cancelled. Filters call `GET /orders` with `?source=` and/or `?status=` (client-side pagination on the result set).
- **Pagination** — **10 orders per page**; footer shows page count and total matching orders.
- **Refresh** — Polls the API every **5 seconds** (same on detail).
- **Empty states** — Hints point at the ingest CLI and runbook when no data or no rows match filters.
- **URL state** — Choosing filters or a page updates `/?source=…&status=…&page=…`. Opening detail keeps that query; **Back to orders** returns to the same filters and page.

### Order detail

- **Header** — Customer name (or **Order {sourceId}** for poll), subtitle from source / poll order # / restaurant (webhook), status badge.
- **Fields** — Total, **Scheduled for** (CSV when scheduled), notes, line items.
- **Poll line items** — Show partner line status (e.g. processing) when present; webhook/CSV lines usually have no line status.
- **Event history** — Scrollable list (`order_created`, `order_updated`, `line_status_changed`, `order_cancelled`, `order_dispatched`) with JSON payloads.
- **Dispatch** — Button when hub status is **`received`** or **`scheduled`**. **`409`** from the API if already dispatched/cancelled.

The UI is **read-only** for ingest except dispatch; loading data is always via the **ingest CLI** or **`POST /webhooks/orders`**.

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
| GET | `/orders` | List orders; optional `?status=` and `?source=webhook\|external_poll\|csv` (combine as needed) |
| GET | `/orders/{id}` | One order by hub UUID |
| GET | `/orders/{id}/events` | Event history for an order |
| POST | `/orders/{id}/dispatch` | Manual dispatch (`received` or `scheduled` only) |
| POST | `/webhooks/orders` | Ingest one webhook JSON body (same shape as one line of `webhook_orders.jsonl`) |

### Examples

List orders:

```bash
curl -s 'http://localhost:8080/orders'
curl -s 'http://localhost:8080/orders?status=received'
curl -s 'http://localhost:8080/orders?source=external_poll'
curl -s 'http://localhost:8080/orders?source=csv&status=scheduled'
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

## Ingest CLI — poll

One jsonl line = one external API poll batch. A **cursor** in SQLite tracks the last fixture line that fully succeeded (HTTP 200, including empty `data`). **500 with partial `data`** saves rows but **does not** advance the cursor (safe to retry the same line).

Run from **`backend/`**:

```bash
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl --limit 10
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl --reset-cursor
go run ./cmd/ingest poll --file ../fixtures/api_responses.jsonl --from 21 --limit 1
```

| Flag | Meaning |
|------|---------|
| `--file PATH` | **Required.** e.g. `../fixtures/api_responses.jsonl` |
| `--from N` | First line (`0` = continue from cursor + 1) |
| `--to N` / `--limit N` | Same as webhook (`--limit` defaults to **1**) |
| `--reset-cursor` | Set cursor to 0 before run |
| `--delay MS` / `-q` | Same as webhook |

Orders use source **`external_poll`** and `sourceId` = poll order number; line items use the fixture hash as `sourceLineId`. Hub status stays **`received`** unless you dispatch/cancel in the UI. Events include **`line_status_changed`** when a line’s status updates (e.g. ordered → processing); **`order_updated`** when new lines attach to an existing poll order.

**Suggested demo:** `--reset-cursor` once, then `--limit 5` twice and open a poll order in the UI to see line statuses and events advance.

---

## Ingest CLI — CSV

Run from **`backend/`** (repeat `--file` for multiple CSVs):

```bash
go run ./cmd/ingest csv --file ../fixtures/orders_1.csv
go run ./cmd/ingest csv --file ../fixtures/orders_1.csv --file ../fixtures/orders_2.csv --limit 20
```

| Flag | Meaning |
|------|---------|
| `--file PATH` | **Required** (repeatable). `orders_1.csv` … `orders_4.csv` |
| `--from N` | First **data** row after header (default `1`) |
| `--to N` / `--limit N` / `--delay MS` / `-q` | Same pattern as webhook |

Bad rows are logged and skipped; exit code **1** if any row failed. Orders use source **`csv`**; identity is a hash of normalized **first name + last name + meal + tomorrow** (same person/meal/slot upserts the same order).

### Scheduling (`scheduled` vs `received`)

Meal maps to a fixed **UTC** time (breakfast **08:00**, lunch **12:00**, dinner **18:00**) on **today** or **tomorrow** from the `tomorrow` column. On each row upsert, if that instant is **after** server `now`, the hub status is **`scheduled`** and **Scheduled for** is set; otherwise **`received`** (no future slot). Status is **not** recomputed when the clock passes—you need another CSV upsert (re-run ingest) to refresh status.

Fixture scale (all four files): **645** data rows → about **236** distinct CSV orders (duplicate survey identities across files produce **`order_updated`** events, not new orders).

**Manual check for CSV `received`:** Re-run CSV ingest after the relevant UTC meal time (e.g. after **08:00 UTC** for “today + breakfast” rows), or filter **source = webhook** / **external_poll** for **`received`** without waiting. See [decisionNotes.md](decisionNotes.md) for scheduling rationale.

### Examples (all files)

```bash
go run ./cmd/ingest csv \
  --file ../fixtures/orders_1.csv \
  --file ../fixtures/orders_2.csv \
  --file ../fixtures/orders_3.csv \
  --file ../fixtures/orders_4.csv \
  -q
```

---

## Data sources (fixtures)

| Source | How to load |
|--------|-------------|
| **Webhook** | `ingest webhook` or `POST /webhooks/orders` |
| **Poll** | `ingest poll --file ../fixtures/api_responses.jsonl` |
| **CSV** | `ingest csv --file ../fixtures/orders_N.csv` |

Files live in [`fixtures/`](../fixtures/):

- `webhook_orders.jsonl` — one JSON order per line (`order_id`, names, `items`, optional `"update": ["cancelled"]`)
- `api_responses.jsonl` — poll mock (`response`, `data`, optional `error`)
- `orders_*.csv` — survey rows (`first_name`, `last_name`, `items`, `notes`, `tomorrow`, `meal`)

Further behavior: [decisionNotes.md](decisionNotes.md).

---

## Troubleshooting

| Issue | What to do |
|-------|------------|
| `go: cannot find main module` | Run `go run` from **`backend/`**, not repo root |
| `address already in use` on 8080 | Stop the old process (`lsof -i :8080`) or change `ROBO_ORDERS_ADDR` |
| UI shows no orders | Ingest sample data; ensure API is on **8080** (Vite proxy default) |
| Ingest ran but UI unchanged | Wait up to 5s for refresh, or check status filter; same DB as API (`backend/data/roboorders.db` by default) |
| List empty but data ingested | Clear **Source** / **Status** filters; check URL query (`?status=scheduled` etc.) |
| All CSV orders `scheduled` | Ingest likely ran while all meal slots were still in the future (UTC); re-ingest after slots pass or see CSV scheduling above |
| Poll ingest “does nothing” | Default **`--limit 1`** per run; run again to advance cursor, or use `--limit N` |
| `SQLITE_BUSY` | Run ingest **sequentially**; avoid parallel ingest against the same DB |
