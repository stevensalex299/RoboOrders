package api

import (
	"errors"
	"net/http"

	"github.com/stevensalex299/RoboOrders/backend/internal/store"
)

func (s *Server) listOrders(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	source := r.URL.Query().Get("source")
	orders, err := s.Store.ListOrders(r.Context(), status, source)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"orders": orders})
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order, err := s.Store.GetOrder(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if order == nil {
		writeErr(w, http.StatusNotFound, errors.New("order not found"))
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) listOrderEvents(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order, err := s.Store.GetOrder(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	if order == nil {
		writeErr(w, http.StatusNotFound, errors.New("order not found"))
		return
	}
	events, err := s.Store.ListOrderEvents(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (s *Server) dispatchOrder(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	order, err := s.Store.DispatchOrder(r.Context(), id)
	if err != nil {
		if errors.Is(err, store.ErrOrderNotFound) {
			writeErr(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, store.ErrNotDispatchable) {
			writeErr(w, http.StatusConflict, err)
			return
		}
		writeErr(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, order)
}
