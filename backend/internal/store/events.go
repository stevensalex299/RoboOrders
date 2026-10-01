package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/stevensalex299/RoboOrders/backend/internal/models"
)

func insertEvent(ctx context.Context, tx *sql.Tx, orderID, typ, payload, at string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO order_events (id, order_id, type, occurred_at, payload)
		VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), orderID, typ, at, payload)
	return err
}

func (s *Store) ListOrderEvents(ctx context.Context, orderID string) ([]models.OrderEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, order_id, type, occurred_at, payload
		FROM order_events WHERE order_id = ? ORDER BY occurred_at ASC`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]models.OrderEvent, 0)
	for rows.Next() {
		var e models.OrderEvent
		var payload string
		if err := rows.Scan(&e.ID, &e.OrderID, &e.Type, &e.OccurredAt, &payload); err != nil {
			return nil, err
		}
		e.Payload = json.RawMessage(payload)
		events = append(events, e)
	}
	return events, rows.Err()
}
