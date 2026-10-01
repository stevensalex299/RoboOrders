# RoboOrders — runbook

Instructions will be filled in as the skeleton is implemented. Target: clone repo, start API + UI, inject orders via mocks.

## Prerequisites

- Go 1.22+ ([install](https://go.dev/dl/))
- Node.js 18+ (frontend uses Vite 6; no Rolldown native bindings)

## Backend (Go API)

From repo root:

```bash
cd backend
go run ./cmd/server
```

Default listen address: `http://localhost:8080` (override with `ROBO_ORDERS_ADDR`).

Health check:

```bash
curl -s http://localhost:8080/health
```

SQLite DB path, migrations, and ingest commands: _coming in next steps._

## Frontend (Vue)

From repo root:

```bash
cd frontend
npm install
npm run dev
```

Dev server URL: _shown in terminal after `npm run dev` (typically http://localhost:5173)._

API requests from the browser use the Vite dev proxy to `localhost:8080` for `/health`, `/orders`, and `/webhooks`.

## Inject sample orders (mocks)

| Source  | Method (planned) |
|---------|------------------|
| Webhook | `POST /webhooks/orders` or CLI replay of `fixtures/webhook_orders.jsonl` |
| Poll    | CLI advancing cursor over `fixtures/api_responses.jsonl` |
| CSV     | CLI over `fixtures/orders_*.csv` |

Details and exact commands will be added when ingest is implemented.

## Fixtures

Sample data is in the repo [`fixtures/`](../fixtures/) directory at the project root.
