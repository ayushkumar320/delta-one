// Package transport exposes the booking service over HTTP.
package transport

import (
	"net/http"

	"github.com/ayush/delta-one/services/booking/internal/service"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// Handler serves the booking HTTP API.
type Handler struct {
	booking *service.Booking
}

// NewHandler returns a Handler for the given service.
func NewHandler(booking *service.Booking) *Handler { return &Handler{booking: booking} }

// Routes returns the service's router with shared middleware applied.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "booking"})
	})

	// Which seats are taken is public: the seat map needs it before sign-in.
	mux.HandleFunc("GET /events/{id}/taken-seats", h.takenSeats)

	mux.Handle("POST /bookings", middleware.RequireUser(http.HandlerFunc(h.hold)))
	mux.Handle("GET /bookings", middleware.RequireUser(http.HandlerFunc(h.list)))
	mux.Handle("GET /bookings/{id}", middleware.RequireUser(http.HandlerFunc(h.get)))
	mux.Handle("POST /bookings/{id}/confirm", middleware.RequireUser(http.HandlerFunc(h.confirm)))
	mux.Handle("DELETE /bookings/{id}", middleware.RequireUser(http.HandlerFunc(h.cancel)))

	mw := append(middleware.Default("booking"), middleware.Identity)
	return middleware.Chain(mux, mw...)
}

func (h *Handler) takenSeats(w http.ResponseWriter, r *http.Request) {
	seats, err := h.booking.TakenSeats(r.Context(), r.PathValue("id"))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"seat_ids": seats})
}

func (h *Handler) hold(w http.ResponseWriter, r *http.Request) {
	var in struct {
		EventID string   `json:"event_id"`
		SeatIDs []string `json:"seat_ids"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	ctx := r.Context()
	booking, err := h.booking.Hold(ctx,
		middleware.UserIDFrom(ctx), middleware.EmailFrom(ctx), in.EventID, in.SeatIDs)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, booking)
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var in struct {
		CardToken string `json:"card_token"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	booking, err := h.booking.Confirm(r.Context(),
		r.PathValue("id"), middleware.UserIDFrom(r.Context()), in.CardToken)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, booking)
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	err := h.booking.Cancel(r.Context(), r.PathValue("id"), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.NoContent(w)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	booking, err := h.booking.Get(r.Context(), r.PathValue("id"), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, booking)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	bookings, err := h.booking.List(r.Context(), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"bookings": bookings})
}
