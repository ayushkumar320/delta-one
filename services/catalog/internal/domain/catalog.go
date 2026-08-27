// Package domain holds the catalog entities: venues, events and the seats an
// event sells. It imports no other layer of the service.
package domain

import (
	"strings"
	"time"

	"github.com/ayush/delta-one/shared/httpx"
)

// Event statuses. Only published events appear in the public listing.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusCancelled = "cancelled"
)

// Venue is a physical place an event happens at.
type Venue struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	City      string    `json:"city"`
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"created_at"`
}

// Event is something with a start time that sells seats at a venue.
type Event struct {
	ID          string    `json:"id"`
	VenueID     string    `json:"venue_id"`
	OrganizerID string    `json:"organizer_id,omitempty"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartsAt    time.Time `json:"starts_at"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`

	// Filled in by list and detail queries, not stored on the row.
	Venue     *Venue `json:"venue,omitempty"`
	SeatCount int    `json:"seat_count"`
	FromCents int64  `json:"from_cents"`
}

// Seat is one sellable place at an event. Whether it is taken lives in the
// booking service; the catalog only describes what exists and what it costs.
type Seat struct {
	ID         string `json:"id"`
	EventID    string `json:"event_id"`
	Section    string `json:"section"`
	RowLabel   string `json:"row_label"`
	SeatNumber int    `json:"seat_number"`
	PriceCents int64  `json:"price_cents"`
}

// Errors the catalog returns.
var (
	ErrEventNotFound = httpx.NotFound("event_not_found", "event not found")
	ErrVenueNotFound = httpx.NotFound("venue_not_found", "venue not found")
)

// MaxPageSize caps how many events one listing request can return, so a client
// cannot ask for the whole table.
const MaxPageSize = 100

// ValidateEvent checks an event before it is stored.
func ValidateEvent(title, venueID string, startsAt time.Time, status string) error {
	if strings.TrimSpace(title) == "" {
		return httpx.BadRequest("missing_title", "title is required")
	}
	if venueID == "" {
		return httpx.BadRequest("missing_venue", "venue_id is required")
	}
	if startsAt.Before(time.Now()) {
		return httpx.BadRequest("past_start", "starts_at must be in the future")
	}
	switch status {
	case StatusDraft, StatusPublished, StatusCancelled:
	default:
		return httpx.BadRequest("invalid_status", "status must be draft, published or cancelled")
	}
	return nil
}

// ValidateSeats checks a seat batch before it is stored.
func ValidateSeats(seats []Seat) error {
	if len(seats) == 0 {
		return httpx.BadRequest("no_seats", "at least one seat is required")
	}
	for _, s := range seats {
		if strings.TrimSpace(s.Section) == "" || strings.TrimSpace(s.RowLabel) == "" {
			return httpx.BadRequest("invalid_seat", "each seat needs a section and a row")
		}
		if s.SeatNumber < 1 {
			return httpx.BadRequest("invalid_seat", "seat_number must be 1 or greater")
		}
		if s.PriceCents < 0 {
			return httpx.BadRequest("invalid_price", "price_cents cannot be negative")
		}
	}
	return nil
}
