// Package client calls the services booking depends on: catalog for seat
// prices and payment for the charge.
package client

import (
	"context"

	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// forward copies the request ID and caller identity onto an outbound call, so
// a downstream service can authorize it and one user action stays traceable
// across every hop.
func forward(ctx context.Context) map[string]string {
	return map[string]string{
		middleware.HeaderRequestID: middleware.RequestIDFrom(ctx),
		middleware.HeaderUserID:    middleware.UserIDFrom(ctx),
		middleware.HeaderUserEmail: middleware.EmailFrom(ctx),
		middleware.HeaderRole:      middleware.RoleFrom(ctx),
	}
}

// Catalog reads event and seat data.
type Catalog struct{ http *httpx.Client }

// NewCatalog returns a Catalog client for a base URL such as
// "http://localhost:8082".
func NewCatalog(baseURL string) *Catalog { return &Catalog{http: httpx.NewClient(baseURL)} }

// Seat is the part of a catalog seat that booking cares about.
type Seat struct {
	ID         string `json:"id"`
	Section    string `json:"section"`
	RowLabel   string `json:"row_label"`
	SeatNumber int    `json:"seat_number"`
	PriceCents int64  `json:"price_cents"`
}

// Event is the part of a catalog event that booking cares about.
type Event struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	StartsAt string `json:"starts_at"`
}

// Event returns one event.
func (c *Catalog) Event(ctx context.Context, eventID string) (Event, error) {
	var event Event
	err := c.http.Do(ctx, "GET", "/events/"+eventID, nil, &event, forward(ctx))
	return event, err
}

// Seats prices a seat selection. Catalog rejects the request when any seat
// does not belong to the event, so booking never holds an unpriced seat.
func (c *Catalog) Seats(ctx context.Context, eventID string, seatIDs []string) ([]Seat, error) {
	body := map[string]any{"seat_ids": seatIDs}
	var out struct {
		Seats []Seat `json:"seats"`
	}
	err := c.http.Do(ctx, "POST", "/events/"+eventID+"/seats/lookup", body, &out, forward(ctx))
	return out.Seats, err
}

// Payment charges bookings.
type Payment struct{ http *httpx.Client }

// NewPayment returns a Payment client for a base URL such as
// "http://localhost:8084".
func NewPayment(baseURL string) *Payment { return &Payment{http: httpx.NewClient(baseURL)} }

// Charge is the outcome of a payment attempt.
type Charge struct {
	ID            string `json:"id"`
	Status        string `json:"status"`
	FailureReason string `json:"failure_reason"`
	AmountCents   int64  `json:"amount_cents"`
}

// Succeeded reports whether the gateway accepted the charge.
func (c Charge) Succeeded() bool { return c.Status == "succeeded" }

// Charge attempts a payment for a booking.
func (p *Payment) Charge(ctx context.Context, bookingID, cardToken string, amountCents int64) (Charge, error) {
	body := map[string]any{
		"booking_id":   bookingID,
		"amount_cents": amountCents,
		"card_token":   cardToken,
	}
	var out Charge
	err := p.http.Do(ctx, "POST", "/payments/charge", body, &out, forward(ctx))
	return out, err
}
