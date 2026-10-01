package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stevensalex299/RoboOrders/backend/internal/models"
)

func (s *Store) DispatchOrder(ctx context.Context, orderID string) (*models.Order, error) {
	order, err := s.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if order.Status != models.StatusReceived && order.Status != models.StatusScheduled {
		return nil, ErrNotDispatchable
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE orders SET status = ?, updated_at = ? WHERE id = ?`,
		models.StatusDispatched, nowStr, orderID)
	if err != nil {
		return nil, err
	}

	robotPayload := buildRobotDispatchPayload(order, nowStr)
	payloadJSON, err := json.Marshal(map[string]any{"robot": robotPayload})
	if err != nil {
		return nil, err
	}
	if err := insertEvent(ctx, tx, orderID, models.EventOrderDispatched, string(payloadJSON), nowStr); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetOrder(ctx, orderID)
}

func buildRobotDispatchPayload(order *models.Order, dispatchedAt string) map[string]any {
	items := make([]map[string]any, 0, len(order.LineItems))
	for _, li := range order.LineItems {
		entry := map[string]any{"name": li.ItemName, "quantity": 1}
		if li.Category != "" {
			entry["category"] = li.Category
		}
		items = append(items, entry)
	}
	payload := map[string]any{
		"version":      1,
		"orderId":      order.ID,
		"source":       order.Source,
		"sourceId":     order.SourceID,
		"priority":     "normal",
		"items":        items,
		"notes":        order.Notes,
		"dispatchedAt": dispatchedAt,
	}
	if order.ScheduledFor != nil && *order.ScheduledFor != "" {
		payload["scheduledFor"] = *order.ScheduledFor
	}
	return payload
}
