// Package transport exposes the payment service over HTTP.
package transport

import (
	"net/http"

	"github.com/ayush/delta-one/services/payment/internal/service"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// Handler serves the payment HTTP API.
type Handler struct {
	payment *service.Payment
}

// NewHandler returns a Handler for the given service.
func NewHandler(payment *service.Payment) *Handler { return &Handler{payment: payment} }

// Routes returns the service's router with shared middleware applied.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "payment"})
	})
	mux.Handle("POST /payments/charge", middleware.RequireUser(http.HandlerFunc(h.charge)))
	mux.Handle("GET /payments/{id}", middleware.RequireUser(http.HandlerFunc(h.get)))

	mw := append(middleware.Default("payment"), middleware.Identity)
	return middleware.Chain(mux, mw...)
}

func (h *Handler) charge(w http.ResponseWriter, r *http.Request) {
	var in struct {
		BookingID   string `json:"booking_id"`
		AmountCents int64  `json:"amount_cents"`
		CardToken   string `json:"card_token"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	// The user is taken from the verified identity, never from the body, so a
	// caller cannot charge a booking on someone else's behalf.
	userID := middleware.UserIDFrom(r.Context())

	payment, err := h.payment.Charge(r.Context(), in.BookingID, userID, in.CardToken, in.AmountCents)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, payment)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	payment, err := h.payment.Get(r.Context(), r.PathValue("id"), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, payment)
}
