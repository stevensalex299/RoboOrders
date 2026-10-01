package models

type Order struct {
	ID            string     `json:"id"`
	Source        string     `json:"source"`
	SourceID      string     `json:"sourceId"`
	FirstName     string     `json:"firstName,omitempty"`
	LastName      string     `json:"lastName,omitempty"`
	Total         *float64   `json:"total,omitempty"`
	Status        string     `json:"status"`
	Notes         string     `json:"notes,omitempty"`
	ScheduledFor  *string    `json:"scheduledFor,omitempty"`
	Restaurant    string     `json:"restaurant,omitempty"`
	OrderPlatform string     `json:"orderPlatform,omitempty"`
	CreatedAt     string     `json:"createdAt"`
	UpdatedAt     string     `json:"updatedAt"`
	LineItems     []LineItem `json:"lineItems,omitempty"`
}

type LineItem struct {
	ID           string   `json:"id"`
	OrderID      string   `json:"orderId,omitempty"`
	ItemName     string   `json:"itemName"`
	SourceLineID string   `json:"sourceLineId,omitempty"`
	Category     string   `json:"category,omitempty"`
	Price        *float64 `json:"price,omitempty"`
	Status       string   `json:"status,omitempty"`
}

const (
	SourceWebhook = "webhook"

	StatusReceived   = "received"
	StatusScheduled  = "scheduled"
	StatusDispatched = "dispatched"
	StatusCancelled  = "cancelled"
)
