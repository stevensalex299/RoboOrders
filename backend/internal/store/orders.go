package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stevensalex299/RoboOrders/backend/internal/models"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) ListOrders(ctx context.Context, status, source string) ([]models.Order, error) {
	q := `SELECT id, source, source_id, first_name, last_name, total, status, notes,
		scheduled_for, restaurant, order_platform, created_at, updated_at
		FROM orders WHERE 1=1`
	args := []any{}
	if status != "" {
		q += ` AND status = ?`
		args = append(args, status)
	}
	if source != "" {
		q += ` AND source = ?`
		args = append(args, source)
	}
	q += ` ORDER BY updated_at DESC`

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]models.Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		items, err := s.lineItemsForOrder(ctx, o.ID)
		if err != nil {
			return nil, err
		}
		o.LineItems = items
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

func (s *Store) GetOrder(ctx context.Context, id string) (*models.Order, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, source, source_id, first_name, last_name, total, status, notes,
		scheduled_for, restaurant, order_platform, created_at, updated_at
		FROM orders WHERE id = ?`, id)
	o, err := scanOrderRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	items, err := s.lineItemsForOrder(ctx, o.ID)
	if err != nil {
		return nil, err
	}
	o.LineItems = items
	return &o, nil
}

type WebhookPayload struct {
	OrderID     string   `json:"order_id"`
	OrderSource string   `json:"order_source"`
	Restaurant  string   `json:"restaurant"`
	FirstName   string   `json:"first_name"`
	LastName    string   `json:"last_name"`
	Total       float64  `json:"total"`
	Items       []string `json:"items"`
	Notes       string   `json:"notes"`
	Update      []string `json:"update"`
}

func (s *Store) UpsertWebhook(ctx context.Context, p WebhookPayload) (*models.Order, error) {
	sourceID := p.OrderID
	if sourceID == "" {
		return nil, fmt.Errorf("order_id required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	status := models.StatusReceived
	cancelled := false
	for _, u := range p.Update {
		if strings.EqualFold(u, "cancelled") {
			cancelled = true
			status = models.StatusCancelled
			break
		}
	}

	var total *float64
	t := p.Total
	total = &t

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID, prevStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT id, status FROM orders WHERE source = ? AND source_id = ?`,
		models.SourceWebhook, sourceID,
	).Scan(&existingID, &prevStatus)

	created := err == sql.ErrNoRows
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	orderID := existingID
	if created {
		orderID = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO orders (
			id, source, source_id, first_name, last_name, total, status, notes,
			restaurant, order_platform, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			orderID, models.SourceWebhook, sourceID, p.FirstName, p.LastName, total, status, p.Notes,
			p.Restaurant, p.OrderSource, now, now,
		)
		if err != nil {
			return nil, err
		}
		payload, _ := json.Marshal(map[string]any{
			"source": models.SourceWebhook, "sourceId": sourceID, "status": status,
		})
		if err := insertEvent(ctx, tx, orderID, models.EventOrderCreated, string(payload), now); err != nil {
			return nil, err
		}
	} else {
		writeStatus := status
		if !cancelled && (prevStatus == models.StatusDispatched || prevStatus == models.StatusCancelled) {
			writeStatus = prevStatus
		}
		_, err = tx.ExecContext(ctx, `UPDATE orders SET
			first_name = ?, last_name = ?, total = ?, status = ?, notes = ?,
			restaurant = ?, order_platform = ?, updated_at = ?
			WHERE id = ?`,
			p.FirstName, p.LastName, total, writeStatus, p.Notes,
			p.Restaurant, p.OrderSource, now, orderID,
		)
		if err != nil {
			return nil, err
		}
		if cancelled {
			payload, _ := json.Marshal(map[string]any{
				"reason": "webhook update cancelled", "previousStatus": prevStatus,
			})
			if err := insertEvent(ctx, tx, orderID, models.EventOrderCancelled, string(payload), now); err != nil {
				return nil, err
			}
		} else {
			evt := map[string]any{
				"fields": []string{"items", "total", "notes"}, "summary": "webhook upsert",
			}
			if writeStatus != status {
				evt["statusPreserved"] = writeStatus
			}
			payload, _ := json.Marshal(evt)
			if err := insertEvent(ctx, tx, orderID, models.EventOrderUpdated, string(payload), now); err != nil {
				return nil, err
			}
		}
	}

	// Sync line items from payload when present, or on non-cancel upserts.
	// Cancel-only updates with no items keep existing line items.
	syncItems := len(p.Items) > 0 || !cancelled
	if syncItems {
		if _, err := tx.ExecContext(ctx, `DELETE FROM line_items WHERE order_id = ?`, orderID); err != nil {
			return nil, err
		}
		for i, name := range p.Items {
			lineID := uuid.NewString()
			sourceLine := fmt.Sprintf("%s:%d", sourceID, i)
			_, err = tx.ExecContext(ctx, `INSERT INTO line_items (id, order_id, item_name, source_line_id)
				VALUES (?, ?, ?, ?)`, lineID, orderID, name, sourceLine)
			if err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetOrder(ctx, orderID)
}

func (s *Store) lineItemsForOrder(ctx context.Context, orderID string) ([]models.LineItem, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, order_id, item_name, source_line_id, category, price, status
		FROM line_items WHERE order_id = ? ORDER BY item_name`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]models.LineItem, 0)
	for rows.Next() {
		var li models.LineItem
		var cat, st sql.NullString
		var price sql.NullFloat64
		if err := rows.Scan(&li.ID, &li.OrderID, &li.ItemName, &li.SourceLineID, &cat, &price, &st); err != nil {
			return nil, err
		}
		if cat.Valid {
			li.Category = cat.String
		}
		if price.Valid {
			v := price.Float64
			li.Price = &v
		}
		if st.Valid {
			li.Status = st.String
		}
		items = append(items, li)
	}
	return items, rows.Err()
}

type scannable interface {
	Scan(dest ...any) error
}

func scanOrder(rows *sql.Rows) (models.Order, error) {
	return scanOrderRow(rows)
}

func scanOrderRow(row scannable) (models.Order, error) {
	var o models.Order
	var fn, ln, notes, sched, rest, platform sql.NullString
	var total sql.NullFloat64
	err := row.Scan(&o.ID, &o.Source, &o.SourceID, &fn, &ln, &total, &o.Status, &notes,
		&sched, &rest, &platform, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return o, err
	}
	if fn.Valid {
		o.FirstName = fn.String
	}
	if ln.Valid {
		o.LastName = ln.String
	}
	if notes.Valid {
		o.Notes = notes.String
	}
	if total.Valid {
		v := total.Float64
		o.Total = &v
	}
	if sched.Valid {
		s := sched.String
		o.ScheduledFor = &s
	}
	if rest.Valid {
		o.Restaurant = rest.String
	}
	if platform.Valid {
		o.OrderPlatform = platform.String
	}
	return o, nil
}
