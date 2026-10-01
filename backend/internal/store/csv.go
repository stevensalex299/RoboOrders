package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/stevensalex299/RoboOrders/backend/internal/models"
)

type CSVRow struct {
	FirstName string
	LastName  string
	Items     []string
	Notes     string
	Tomorrow  bool
	Meal      string
}

func CSVSourceID(row CSVRow) string {
	norm := func(s string) string { return strings.ToLower(strings.TrimSpace(s)) }
	key := norm(row.FirstName) + "|" + norm(row.LastName) + "|" + norm(row.Meal) + "|" + fmt.Sprintf("%t", row.Tomorrow)
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func mealSlotUTC(meal string, tomorrow bool, now time.Time) time.Time {
	now = now.UTC()
	hour := 12
	switch strings.ToLower(strings.TrimSpace(meal)) {
	case "breakfast":
		hour = 8
	case "lunch":
		hour = 12
	case "dinner":
		hour = 18
	}
	y, m, d := now.Date()
	if tomorrow {
		d++
	}
	return time.Date(y, m, d, hour, 0, 0, 0, time.UTC)
}

func (s *Store) UpsertCSVRow(ctx context.Context, row CSVRow) (*models.Order, error) {
	sourceID := CSVSourceID(row)
	now := time.Now().UTC()
	slot := mealSlotUTC(row.Meal, row.Tomorrow, now)
	nowStr := now.Format(time.RFC3339)

	status := models.StatusReceived
	var scheduledFor *string
	if slot.After(now) {
		status = models.StatusScheduled
		s := slot.Format(time.RFC3339)
		scheduledFor = &s
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var existingID, prevStatus string
	err = tx.QueryRowContext(ctx,
		`SELECT id, status FROM orders WHERE source = ? AND source_id = ?`,
		models.SourceCSV, sourceID,
	).Scan(&existingID, &prevStatus)

	created := err == sql.ErrNoRows
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	orderID := existingID
	writeStatus := status
	if !created {
		if prevStatus == models.StatusDispatched || prevStatus == models.StatusCancelled {
			writeStatus = prevStatus
		}
	}

	if created {
		orderID = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO orders (
			id, source, source_id, first_name, last_name, status, notes, scheduled_for, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			orderID, models.SourceCSV, sourceID, row.FirstName, row.LastName, writeStatus, row.Notes, scheduledFor, nowStr, nowStr,
		)
		if err != nil {
			return nil, err
		}
		payload, _ := json.Marshal(map[string]any{
			"source": models.SourceCSV, "sourceId": sourceID, "status": writeStatus, "scheduledFor": scheduledFor,
		})
		if err := insertEvent(ctx, tx, orderID, models.EventOrderCreated, string(payload), nowStr); err != nil {
			return nil, err
		}
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE orders SET
			first_name = ?, last_name = ?, status = ?, notes = ?, scheduled_for = ?, updated_at = ?
			WHERE id = ?`,
			row.FirstName, row.LastName, writeStatus, row.Notes, scheduledFor, nowStr, orderID,
		)
		if err != nil {
			return nil, err
		}
		payload, _ := json.Marshal(map[string]any{
			"fields": []string{"items", "notes", "scheduledFor"}, "summary": "csv upsert",
		})
		if err := insertEvent(ctx, tx, orderID, models.EventOrderUpdated, string(payload), nowStr); err != nil {
			return nil, err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM line_items WHERE order_id = ?`, orderID); err != nil {
		return nil, err
	}
	for i, name := range row.Items {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		lineID := uuid.NewString()
		sourceLine := fmt.Sprintf("%s:%d", sourceID, i)
		if _, err := tx.ExecContext(ctx, `INSERT INTO line_items (id, order_id, item_name, source_line_id)
			VALUES (?, ?, ?, ?)`, lineID, orderID, name, sourceLine); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetOrder(ctx, orderID)
}

func ParseCSVItemsField(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.Contains(raw, "\n") {
		lines := strings.Split(raw, "\n")
		out := make([]string, 0, len(lines))
		for _, ln := range lines {
			ln = strings.TrimSpace(ln)
			if ln != "" {
				out = append(out, ln)
			}
		}
		return out
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
