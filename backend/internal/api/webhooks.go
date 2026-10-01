package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

func (s *Server) webhookOrders(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	var p store.WebhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	order, err := s.Store.UpsertWebhook(r.Context(), p)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}
