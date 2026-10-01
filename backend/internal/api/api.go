package api

import (
	"net/http"

	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

type Server struct {
	Store *store.Store
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /orders", s.listOrders)
	mux.HandleFunc("GET /orders/{id}/events", s.listOrderEvents)
	mux.HandleFunc("POST /orders/{id}/dispatch", s.dispatchOrder)
	mux.HandleFunc("GET /orders/{id}", s.getOrder)
	mux.HandleFunc("POST /webhooks/orders", s.webhookOrders)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
