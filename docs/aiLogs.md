# AI Session log - RoboOrders take home

## How I used AI on this assignment

My plan for AI usage in this assignment is to make sure I fully understand the assignment itself and design/architect it myself. I made it a point to strictly use AI for claryfing any questions about writeup verbiage and any boilerplate code that would add an extensive amount of time to hand write.

My goal is to leverage existing AI tools, while maintaining my self worked plan. I may also bounce ideas back and forth if necessary.

## Session 1 — Design & Fixtures (2026-09-30)

- Pasted the assignment — what are the scope and deliverables?
- Do ingest paths create orders in our store or only display them? For polling, is that our DB or an external upstream?
- Can orders overlap across webhook / poll / CSV, or should they stay separate per source?
- Does a CLI ingest path outside Vue fit the assignment?
- Review decisionNotes against the assignment — anything missing before we implement?
- Help me tighten the Ingest section (fixtures, time_since/cursor); I edited suggestions before keeping them.

## Session 2 — Project scaffold (2026-09-30)

- Scaffold Go API and Vue frontend from decisionNotes health check only for the first slice.
- Add Tailwind CSS, remove unused Vite template files, and keep the dev proxy aimed at upcoming /orders routes.
- I want a root README that points reviewers to docs/runbook.md for run instructions.

## Session 3 — Webhook ingest, API, list UI (2026-09-30)

- Implement the webhook slice from decisionNotes: SQLite schema + embedded migrate, orders store with upsert/cancel on (source, sourceId), order events and line items.
- Scaffold the Go backend layout (cmd/server, cmd/ingest, internal/api, internal/store, internal/models) — I'm learning Go, keep idiomatic but straightforward.
- Expose GET /health, GET /orders with optional status filter, GET /orders/{id}, POST /webhooks/orders with CORS for the Vue dev server.
- Build a sequential webhook ingest CLI (--file, --from, --to, --limit, --delay, DB mode and --http); document commands in docs/runbook.md for reviewers.
- Wire the Vue list: table columns from the API, status dropdown, poll GET /orders every 5s via the Vite proxy.
- I hit SQLITE_BUSY on parallel/burst ingest — remove burst/concurrency and keep sequential-only for this take-home.
- Give me a runbook section with curl examples so I can manually verify health, list, filters, get-by-id, webhook POST, and cancel before I sign off.

## Session 4 — Order detail, dispatch, webhook upsert fixes (2026-09-30)

- Implement detail + dispatch from decisionNotes: order events model/store, GET /orders/{id}/events, POST /orders/{id}/dispatch with robot payload on order_dispatched and 409 when not dispatchable.
- Split internal/api into api.go, orders.go, webhooks.go, response.go; keep handlers thin over the store.
- Add vue-router, App shell, OrderList + OrderDetail, OrderEventsList, shared format helpers and usePolling on detail (same 5s refresh as the list).
- Fix cancel webhooks that still send items (fixture Jamie line 64): sync line items when len(items) > 0 or the update is not cancel-only so cancelled orders keep/show lines.
- Fix Vite dev hard refresh on /orders/{id} — proxy bypass serves index.html when Accept includes text/html so the SPA loads instead of raw JSON.
- After dispatch or cancel, non-cancel webhook upserts should not reset hub status to received: preserve dispatched/cancelled, still update fields and order_updated; optional statusPreserved in event payload; document in decisionNotes Identity & Upsert.
- Trim runbook to reviewer-focused run/ingest/UI/API; drop internal QA checklists — I'll delete DB and retest end-to-end before submit.
