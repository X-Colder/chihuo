package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

func (s *Server) handleCreatePaymentIntent(w http.ResponseWriter, r *http.Request) {
	current := mustPrincipal(r)
	body, err := readJSONBody(r)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	var probe struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(body, &probe); err != nil {
		s.writeOperationError(w, r, newRequestError(http.StatusBadRequest, "INVALID_JSON", "request body is invalid", nil))
		return
	}
	order, err := s.store.GetOrder(r.Context(), probe.OrderID)
	if err != nil {
		s.writeOperationError(w, r, err)
		return
	}
	if order.ConsumerID != current.ID {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "order does not belong to the current user", nil)
		return
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	s.paymentHandler.HandleCreatePaymentIntent(w, r)
}
