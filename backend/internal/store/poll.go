package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/stevensalex299/RoboOrders/backend/internal/models"
)

type PollBatchLine struct {
	Response int
	Error    string
	Data     map[string]PollLineItem
}

type PollLineItem struct {
	Order    int
	Name     string
	Category string
	Price    float64
	Status   string
}

type PollApplyResult struct {
	AdvanceCursor bool
}

func ParsePollBatchJSON(body []byte) (PollBatchLine, error) {
	var raw struct {
		Response int                       `json:"response"`
		Error    string                    `json:"error"`
		Data     map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return PollBatchLine{}, err
	}
	out := PollBatchLine{
		Response: raw.Response,
		Error:    raw.Error,
		Data:     make(map[string]PollLineItem, len(raw.Data)),
	}
	for hash, blob := range raw.Data {
		var item struct {
			Order    int     `json:"order"`
			Name     string  `json:"name"`
			Category string  `json:"category"`
			Price    float64 `json:"price"`
			Status   string  `json:"status"`
		}
		if err := json.Unmarshal(blob, &item); err != nil {
			return PollBatchLine{}, fmt.Errorf("data[%s]: %w", hash, err)
		}
		out.Data[hash] = PollLineItem{
			Order: item.Order, Name: item.Name, Category: item.Category,
			Price: item.Price, Status: item.Status,
		}
	}
	return out, nil
}

func (s *Store) ApplyPollBatch(ctx context.Context, batch PollBatchLine) (PollApplyResult, error) {
	advance := false
	switch batch.Response {
	case 200:
		advance = true
	case 500:
		if len(batch.Data) == 0 {
			return PollApplyResult{}, fmt.Errorf("response 500 with no data")
		}
		advance = false
	default:
		return PollApplyResult{}, fmt.Errorf("unsupported response code %d", batch.Response)
	}

	if len(batch.Data) == 0 {
		return PollApplyResult{AdvanceCursor: advance}, nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return PollApplyResult{}, err
	}
	defer tx.Rollback()

	affectedOrders := make(map[string]struct{})
	touchedOrders := make(map[string]struct{})

	for hash, item := range batch.Data {
		sourceID := strconv.Itoa(item.Order)
		orderID, created, err := ensurePollOrder(ctx, tx, sourceID, now)
		if err != nil {
			return PollApplyResult{}, err
		}
		touchedOrders[orderID] = struct{}{}
		if created {
			payload, _ := json.Marshal(map[string]any{
				"source": models.SourceExternalPoll, "sourceId": sourceID, "status": models.StatusReceived,
			})
			if err := insertEvent(ctx, tx, orderID, models.EventOrderCreated, string(payload), now); err != nil {
				return PollApplyResult{}, err
			}
		}

		lineID, prevLineStatus, err := lookupLineBySourceLine(ctx, tx, orderID, hash)
		if err != nil {
			return PollApplyResult{}, err
		}

		if lineID == "" {
			lineID = uuid.NewString()
			_, err = tx.ExecContext(ctx, `INSERT INTO line_items (id, order_id, item_name, source_line_id, category, price, status)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				lineID, orderID, item.Name, hash, item.Category, item.Price, item.Status)
			if err != nil {
				return PollApplyResult{}, err
			}
			if !created {
				affectedOrders[orderID] = struct{}{}
			}
		} else {
			statusChanged := prevLineStatus != "" && prevLineStatus != item.Status
			_, err = tx.ExecContext(ctx, `UPDATE line_items SET item_name = ?, category = ?, price = ?, status = ? WHERE id = ?`,
				item.Name, item.Category, item.Price, item.Status, lineID)
			if err != nil {
				return PollApplyResult{}, err
			}
			if statusChanged {
				payload, _ := json.Marshal(map[string]any{
					"sourceLineId": hash, "from": prevLineStatus, "to": item.Status, "itemName": item.Name,
				})
				if err := insertEvent(ctx, tx, orderID, models.EventLineStatusChanged, string(payload), now); err != nil {
					return PollApplyResult{}, err
				}
			}
		}
	}

	for orderID := range touchedOrders {
		if err := refreshPollOrderTotal(ctx, tx, orderID, now); err != nil {
			return PollApplyResult{}, err
		}
	}

	for orderID := range affectedOrders {
		payload, _ := json.Marshal(map[string]any{
			"fields": []string{"lineItems", "total"}, "summary": "poll batch",
		})
		if err := insertEvent(ctx, tx, orderID, models.EventOrderUpdated, string(payload), now); err != nil {
			return PollApplyResult{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return PollApplyResult{}, err
	}
	return PollApplyResult{AdvanceCursor: advance}, nil
}

func ensurePollOrder(ctx context.Context, tx *sql.Tx, sourceID, now string) (orderID string, created bool, err error) {
	err = tx.QueryRowContext(ctx,
		`SELECT id FROM orders WHERE source = ? AND source_id = ?`,
		models.SourceExternalPoll, sourceID,
	).Scan(&orderID)
	if err == sql.ErrNoRows {
		orderID = uuid.NewString()
		_, err = tx.ExecContext(ctx, `INSERT INTO orders (
			id, source, source_id, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)`,
			orderID, models.SourceExternalPoll, sourceID, models.StatusReceived, now, now,
		)
		return orderID, true, err
	}
	return orderID, false, err
}

func lookupLineBySourceLine(ctx context.Context, tx *sql.Tx, orderID, sourceLineID string) (lineID, status string, err error) {
	err = tx.QueryRowContext(ctx,
		`SELECT id, COALESCE(status, '') FROM line_items WHERE order_id = ? AND source_line_id = ?`,
		orderID, sourceLineID,
	).Scan(&lineID, &status)
	if err == sql.ErrNoRows {
		return "", "", nil
	}
	return lineID, status, err
}

func refreshPollOrderTotal(ctx context.Context, tx *sql.Tx, orderID, now string) error {
	var prevStatus string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM orders WHERE id = ?`, orderID).Scan(&prevStatus); err != nil {
		return err
	}
	var total sql.NullFloat64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(price), 0) FROM line_items WHERE order_id = ?`, orderID,
	).Scan(&total); err != nil {
		return err
	}
	writeStatus := prevStatus
	if prevStatus != models.StatusDispatched && prevStatus != models.StatusCancelled {
		writeStatus = models.StatusReceived
	}
	var t *float64
	if total.Valid {
		v := total.Float64
		t = &v
	}
	_, err := tx.ExecContext(ctx, `UPDATE orders SET total = ?, status = ?, updated_at = ? WHERE id = ?`,
		t, writeStatus, now, orderID)
	return err
}
