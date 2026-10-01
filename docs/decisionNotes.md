# Decision & Architectural Notes

Below I have included my thought process into the assignment, why I designed it the way I did, etc.

## Stack

I have chosen to go forward with a Vue.js frontend and Go backend. Not only does the team already use these stacks, this assignment will give me a chance to interact with this stack before potentially working at the company. Also, it allows me to see how I can leverage the advantages of Go and Vue for a robot ordering system.

For databasing, I can use SQLlite and auto schema or docker to spin up the necessary components for testing this application.

## Orders By Source

The mocks provide examples of orders with a variety of differences depending on the actual source.

### API Polling

The API responses consist of an object with the response code for the response. If the response is invalid, like a 500, it may also consist of a partial data object, but there is also an error message. For responses with data, the data can be an empty object or is in the form of a map of objects with a hash key per line item. Items in the same order have the same order number.

Order objects have the following:

- order (order number) (will be source id)
- name
- category
- price
- status (enum most likely)

Categories in the mock include:

- Beverage
- Coffee & Tea
- Toasts & Light Bites
- Side
- Breakfast (All Day)
- Hot Sandwiches
- Cold Sandwiches
- Chicken
- Appetizers
- Noodles & Rice
- Griddle
- Lunch & Dinner
- Entrees (with accent)
- Desserts

Status in the mock include:

- ordered
- processing
- with_courier
- delivered

### CSV

The CSV order files consist of the following fields:

- first_name
- last_name
- items
- notes
- tomorrow
- meal

Items can potentially be a list of items, starting and ending with "". It can either be comma separated for each individual item like "chicken, bacon, toast" or each separate item is on a different line without commas.

Notes is just a line of text without apostrophes, often empty.

Tomorrow I believe means if the meal is scheduled for today or tomorrow, true being scheduled for tomorrow.

Meal consists of the following options:

- breakfast
- lunch
- dinner

I'm going to move forward with the idea that it means what time of day the item is scheduled for.

### Webhook

The webhook file consists of a line by line list of orders. Each order has the following fields:

- order_id (UUID) (will be source id)
- order_source
- restaurant
- first_name
- last_name
- total
- items (list of strings)
- notes
- update (optional cancel)

Seems fairly organized and would be easy to parse.

## Data Handling

There are three methods that data can ingest data including the webhook, csv, and api polling. All of which have the capacity to potentially update, cancel, or create new data. Carefully considering this process is key to scaling and managing an application that can handle orders successfully.

### Maintaining & Defining Data

The biggest part of the ingesting is properly managing and creating data from these ingest sources. I maintain a common list of orders in SQLite (source of truth) as data is ingested. The Vue UI loads orders via the API with a shared polling composable on list and detail views.

On our backend, I will merge incoming orders to a strict format so that, regardless of the source, I can easily show and display orders on the frontend. I will also keep the orders distinct based on the source they are coming from, meaning an order from webhook is unique from an order coming from the api poll (no way to correlate them currently). The order format should look similar to the following to properly handle all of our ingest sources:

Order (Unique Key on source, sourceId)
{
id: UUID
source: OrderSourceEnum
sourceId: string
firstName: string?
lastName: string?
total: float64?
items: OrderLineItem[]
status: OrderStatusEnum
notes: string?
scheduledFor: Timestamp?
metadata: OrderSourceMetadata
createdAt: Timestamp
updatedAt: Timestamp
}

OrderSourceMetadata
{
restaurant: string?
orderPlatform: string?
}

OrderLineItem
{
itemName: string
sourceLineId: string?
category: string?
price: float64?
status: LineItemStatusEnum?
}

OrderEvent (for historical events)
{
id: UUID
orderID: UUID
type: OrderEventType
occurredAt: Timestamp
payload: OrderEventPayload
}

OrderSourceEnum (stored values): "webhook" | "external_poll" | "csv"
OrderStatusEnum: "received" | "scheduled" | "dispatched" | "cancelled"
LineItemStatusEnum: "ordered" | "processing" | "with_courier" | "delivered"
OrderEventType: "order_created" |"order_updated" | "line_status_changed" | "order_cancelled" |"order_dispatched"

OrderEventPayload is shaped by the event type:
order_created: { source, sourceId, status, scheduledFor? }
order_updated: { fields: string[], summary?: string, statusPreserved?: string }
line_status_changed: { sourceLineId, from, to, itemName? }
order_cancelled: { reason, previousStatus }
order_dispatched: { robot: RobotDispatchPayload }

RobotDispatchPayload
{
version: number
orderId: UUID
source: OrderSourceEnum
sourceId: string
scheduledFor: Timestamp?
priority: string
items: { name: string, quantity: number, category?: string }[]
notes: string?
dispatchedAt: Timestamp
}

Each order will be merged correctly into a singular order item as they are ingested. The source of ingestion will be defined, alongside any unique id that source provided (CSV will define it's own id). The source and source id provide a way to distinguish between updating an existing order or creating a new one.

I also wanted to capture the commonalities between ingest source data in the order object such as order name. Any distinct items were added as metadata or optional attributes under specific order line items.

When it comes down to statuses, I wanted to capture the difference between line-level statuses like with_courier (that are only included in the polling) and the hub-level statuses (such as dispatched, scheduled, cancelled, etc). I also have historical enums for the set of events.

### Parsing Data

Parsing differs by ingest source. cmd/ingest provides webhook, poll, and csv subcommands against the fixture files; the frontend only reads the API and shows orders and event history.

For parsing poll data, we will need to consider whether the api call was a failure, partial failure, or successful. If data was returned, we should always save the order on our end, whether the response is a 500 or not. We don't want to lose orders. In terms of showing errors, we can still make note of the error message through logging so we can track them. Each poll entry has a hash, which we should utilize to know whether line items were updated or created. As we ingest, we will need to add appropriate order events. Status for poll orders has a bit more detail down to the line item, so we can show this info and fall back for other ingest methods to the overall order status.

Parsing webhook and csv data will be a bit simpler. For webhook data, we parse each line and create an order per line. We can also create a line item for each item in the list. Since we don't have detailed status here for line items, we fall back to using the overall order statuses. CSV works very similarly, except there is a bit more line item parsing based on the different way it shows order items.

### Backend Storage

All ingest sources share one order shape in SQLite: orders, line_items, order_events, plus ingest_state for the poll cursor. There is a unique constraint on (source, source_id) and indexes on status, updated time, and (order_id, source_line_id) for line upserts.

## Application Lifecycle

In this section I want to outline the overall lifecycle of the application and how pieces interact with each other.

### Identity & Upsert

As orders are ingested from our various ingest sources, it's important to know the difference between existing and non-existing orders. By utilizing the distinct relationship between our source/sourceId, we can do just that. However, for new orders, we need to define that.

Since there isn't a great way to know whether an order from a poll is identical to an order from the webhook or CSV, I decided to keep them distinct. We can use those as our source for each order. Source Id is defined based on the ingest method. Api polling has an order number we can use for source id and a hash we can use for the individual line item. Webhook has the order id. CSV doesn't have an identifiable id, so we can determine one by hashing the order line (such as a hash of normalized first name, last name, meal, and tomorrow value, so meal/tomorrow changes determine new or existing order).

When our source/source id combo is first encountered on ingest, we will create the order and record the order_created event. When encountered again, we will update and record the order_updated event. For any line item updates (api polling), we will make the necessary line item updates and record the appropriate event. If the parent order hasn't been created, we will create it as well. In the case of webhooks, if we receive the update cancelled, we will appropriately update the hub status of that order.

If hub status is already dispatched (manual dispatch) or cancelled, a later webhook line (that isn't cancelling the order) still refreshes order fields and line items and records order_updated, but status stays dispatched or cancelled instead of reverting to received. A cancel webhook always sets cancelled and records order_cancelled. When status is preserved, order_updated may include statusPreserved in the payload. This covers partner retries and out-of-order updates without undoing a dispatch or cancel in our hub.

### Ingest

Ingest uses CLI commands against the fixture files. Data lands in the same SQLite file the API uses so the UI can list, filter, and drill into orders and events.

For polling specifically, our fixture takes the place of the external API we would call with a 'time_since' query parameter.

We keep a poll cursor in the backend, such as the index of the last applied line in api_responses.jsonl. Each poll run applies the next line's data and only advances the cursor when that batch succeeds. Empty data on a 200 still counts as success. If the response is an error or 500 but includes partial data, we still save that data and leave the cursor unchanged so a retry does not duplicate rows. Upserts by order number and line hash keep replays idempotent.

Webhook ingest uses webhook_orders.jsonl (one line per order) and POST /webhooks/orders for the same payload shape. CSV ingest uses orders_1.csv through orders_4.csv. Those files may be uploaded more than once per day and we upsert on source and sourceId. Re-uploading the same survey row updates the same order (hash of normalized first name, last name, meal, and tomorrow). Item or note changes on that row update the existing order instead of creating a new one.

Our webhook and CSV ingest methods produce entire orders per line or row, using hub-level statuses. The external poll ingest applies one response batch at a time and groups line items under a parent order.

The poll fixtures show new lines and status updates but not explicit cancellations. If the partner API sent a cancel, we would set hub status to cancelled and record an order_cancelled event, like webhook update cancelled. Scheduled orders from the poll API are not in the sample data; hub scheduled is driven by CSV meal and tomorrow.

### Dispatch

For the purpose of this assignment, dispatching can be done manually using a UI trigger on an order that has been received or scheduled. The order status will then be updated to dispatched and a skeleton payload will be saved on the event object.

### Scheduling

Currently in the fixture data, only the CSV fixtures carry scheduling hints. For this assignment, I took the tomorrow and meal fields to indicate when the order is scheduled for. The meal field (breakfast, lunch, dinner) will represent fixed times of the day and tomorrow determines if the order is for today or tomorrow. From these fields we can determine the order scheduledFor field, comparing to the current server time. If the slot is still in the future, we will set the order as scheduled, otherwise the order is set as received and our scheduledFor is cleared. For now, our webhook and external poll will remain as realtime orders, but future support could be added to allow scheduling during ingest.

## UI Layer

The frontend loads orders from the Go API. SQLite is the source of truth. The list screen supports filters on source and status, shows line counts and totals, and links each row to a detail view. Poll orders without customer names display as Order {sourceId}. Detail shows metadata, line items (including line-level status for poll data), scheduledFor when set, event history, and manual dispatch for received or scheduled. List and detail refresh every 5s via a shared composable. Styling uses Tailwind CSS for layout and status badges.

## API Layer

Our Go api layer will effectively control how we read data into our frontend layer. We will have the following read endpoints:

- GET /health (health of the server)
- GET /orders (list; optional query `status` and `source`)
- GET /orders/:id (gets the specified order)
- GET /orders/:id/events (gets the history for an order)

The api will also have a few post actions for manual dispatch and invoking the webhook ingest:

- POST /orders/:id/dispatch
- POST /webhooks/orders

## Scale & Fault Tolerance

Although we aren't planning for full production scale, we still need to consider how we might handle 100,000 requests a day. For this particular skeleton, using Go and SQL with indexes on the source/sourceId fields, alongside proper statuses, should be a good start to handle that amount of traffic. Any bursty traffic, such as from webhooks, would be the big thing to look out for. If we wanted to scale out even further, we could add a prioritization queue, other app instances, or utilize helpful database features such as Postgres read replicas.

In regards to fault tolerance, our ingest paths should fail predictably and safely. Our webhook ingest will upsert on order_id so that retries and bursts don't duplicate orders. For polling, we can use a cursor and only advance if batches succeed. CSV processing should skip bad rows and continue the file, this way one bad line doesn't fail the whole upload. Any odd responses can be logged and our order events will be kept for audit purposes.

## Next Steps

- Hub cancel and schedule by ingest (today): Webhook implements hub cancel (update: cancelled = order_cancelled). CSV implements hub schedule (meal + tomorrow = scheduledFor / scheduled vs received). Poll implements line-level deltas (new lines, status changes, partial 500) on realtime hub orders (the poll mock does not include hub cancel or hub scheduled payloads). If a live poll API sent those, we would set the same hub statuses and events as webhook/CSV rather than inventing a separate model. CSV survey rows have no cancel field—re-upload updates the same order, explicit CSV cancel would be a future convention if the form added one. I would also base the distinction for our fixed schedule meal times off local time from where the order was scheduled.
- GET /orders time range/pagination — filter by updatedAt or createdAt window for large datasets. Also adding pagination will make queries faster.
- Pinia (or similar) — central client cache if the UI grows beyond list + detail.
- Live poll HTTP — replace fixture jsonl with a real time_since partner API. Keep the same cursor and batch apply logic (today the cursor over api_responses.jsonl stands in for time_since).
- Poll hub cancel — if the partner API sends cancellations, map to cancelled + order_cancelled (fixtures do not include this today).
- Scheduling on webhook/poll ingest — CSV already drives scheduled. Extend only if product needs it.
- Production scale — queue for webhook bursts, Postgres/read replicas, multi-instance deploy (outlined under Scale & Fault Tolerance). 100k requests/day can be appropriately handled with the use of Go and indexed SQLite upserts, these production changes will advance that.
- Full suite of Unit/Functional tests (testing was done through manual curls and inspecting).
- Automated hub lifecycle - Background job or partner events to transition order statuses. This may include transitioning from scheduled to received, dispatched to a final state after order is completed, etc.
