// Package transport exposes the catalog service over HTTP.
package transport

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ayush/delta-one/services/catalog/internal/domain"
	"github.com/ayush/delta-one/services/catalog/internal/repository"
	"github.com/ayush/delta-one/services/catalog/internal/service"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// Handler serves the catalog HTTP API.
type Handler struct {
	catalog *service.Catalog
}

// NewHandler returns a Handler for the given service.
func NewHandler(catalog *service.Catalog) *Handler { return &Handler{catalog: catalog} }

// Routes returns the service's router with shared middleware applied.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "catalog"})
	})

	// Browsing is public.
	mux.HandleFunc("GET /events", h.listEvents)
	mux.HandleFunc("GET /events/{id}", h.getEvent)
	mux.HandleFunc("GET /events/{id}/seats", h.listSeats)
	mux.HandleFunc("GET /venues", h.listVenues)

	// The booking service prices a hold through this endpoint.
	mux.HandleFunc("POST /events/{id}/seats/lookup", h.lookupSeats)

	// Publishing is limited to organizers.
	organizer := middleware.RequireRole("organizer")
	mux.Handle("POST /events", organizer(http.HandlerFunc(h.createEvent)))
	mux.Handle("POST /venues", organizer(http.HandlerFunc(h.createVenue)))

	mw := append(middleware.Default("catalog"), middleware.Identity)
	return middleware.Chain(mux, mw...)
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))

	events, err := h.catalog.ListEvents(r.Context(), repository.EventFilter{
		City:   q.Get("city"),
		Search: q.Get("q"),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) getEvent(w http.ResponseWriter, r *http.Request) {
	event, err := h.catalog.Event(r.Context(), r.PathValue("id"), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, event)
}

func (h *Handler) listSeats(w http.ResponseWriter, r *http.Request) {
	seats, err := h.catalog.Seats(r.Context(), r.PathValue("id"), middleware.UserIDFrom(r.Context()))
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"seats": seats})
}

func (h *Handler) lookupSeats(w http.ResponseWriter, r *http.Request) {
	var in struct {
		SeatIDs []string `json:"seat_ids"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	seats, err := h.catalog.SeatsByIDs(r.Context(), r.PathValue("id"), in.SeatIDs)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"seats": seats})
}

func (h *Handler) listVenues(w http.ResponseWriter, r *http.Request) {
	venues, err := h.catalog.Venues(r.Context())
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"venues": venues})
}

func (h *Handler) createVenue(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string `json:"name"`
		City    string `json:"city"`
		Address string `json:"address"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}
	venue, err := h.catalog.CreateVenue(r.Context(), in.Name, in.City, in.Address)
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, venue)
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		VenueID     string    `json:"venue_id"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		StartsAt    time.Time `json:"starts_at"`
		Status      string    `json:"status"`
		Seats       []struct {
			Section    string `json:"section"`
			RowLabel   string `json:"row_label"`
			SeatNumber int    `json:"seat_number"`
			PriceCents int64  `json:"price_cents"`
		} `json:"seats"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		httpx.Fail(w, err)
		return
	}

	seats := make([]domain.Seat, len(in.Seats))
	for i, s := range in.Seats {
		seats[i] = domain.Seat{
			Section: s.Section, RowLabel: s.RowLabel,
			SeatNumber: s.SeatNumber, PriceCents: s.PriceCents,
		}
	}

	event, err := h.catalog.CreateEvent(r.Context(), middleware.UserIDFrom(r.Context()), service.NewEvent{
		VenueID:     in.VenueID,
		Title:       in.Title,
		Description: in.Description,
		StartsAt:    in.StartsAt,
		Status:      in.Status,
		Seats:       seats,
	})
	if err != nil {
		httpx.Fail(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, event)
}
