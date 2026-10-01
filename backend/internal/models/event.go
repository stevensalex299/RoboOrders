package models

import "encoding/json"

type OrderEvent struct {
	ID         string          `json:"id"`
	OrderID    string          `json:"orderId"`
	Type       string          `json:"type"`
	OccurredAt string          `json:"occurredAt"`
	Payload    json.RawMessage `json:"payload"`
}

const (
	EventOrderCreated      = "order_created"
	EventOrderUpdated      = "order_updated"
	EventLineStatusChanged = "line_status_changed"
	EventOrderCancelled    = "order_cancelled"
	EventOrderDispatched   = "order_dispatched"
)
