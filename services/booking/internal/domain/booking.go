// Package domain holds the booking entities and the rules that govern a seat
// hold. It imports no other layer of the service.
package domain

import (
	"time"

	"github.com/ayush/delta-one/shared/httpx"
)

// Booking statuses.
const (
	StatusHeld      = "held"
	StatusConfirmed = "confirmed"
	StatusCancelled = "cancelled"
	StatusExpired   = "expired"
)

// HoldTTL is how long seats stay reserved before payment. Long enough to enter
// card details, short enough that abandoned checkouts return seats to the pool.
const HoldTTL = 10 * time.Minute

// MaxSeatsPerBooking caps one booking, which is the cheapest defence against a
// single request holding an entire venue.
const MaxSeatsPerBooking = 8

// Seat is one seat within a booking, priced when the hold was taken so a later
// price change does not alter what the user agreed to pay.
type Seat struct {
	SeatID     string `json:"seat_id"`
	PriceCents int64  `json:"price_cents"`
}

// Booking is a user's claim on seats for an event.
type Booking struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	UserEmail     string    `json:"user_email"`
	EventID       string    `json:"event_id"`
	EventTitle    string    `json:"event_title"`
	Status        string    `json:"status"`
	TotalCents    int64     `json:"total_cents"`
	HoldExpiresAt time.Time `json:"hold_expires_at"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Seats         []Seat    `json:"seats"`
}

// Expired reports whether a held booking's window has passed. A booking is
// only expired while it is still held; confirming ends the countdown.
func (b Booking) Expired(now time.Time) bool {
	return b.Status == StatusHeld && now.After(b.HoldExpiresAt)
}

// Errors the booking service returns.
var (
	ErrBookingNotFound = httpx.NotFound("booking_not_found", "booking not found")
	ErrSeatsTaken      = httpx.Conflict("seats_taken",
		"one or more of those seats were just taken; pick different seats")
	ErrHoldExpired = httpx.Conflict("hold_expired",
		"your seat hold expired; start again to pick seats")
	ErrNotHeld          = httpx.Conflict("not_held", "this booking is no longer awaiting payment")
	ErrAlreadyConfirmed = httpx.Conflict("already_confirmed", "this booking is already confirmed")
)

// ValidateHold checks a hold request before any database work happens.
func ValidateHold(eventID string, seatIDs []string) error {
	if eventID == "" {
		return httpx.BadRequest("missing_event", "event_id is required")
	}
	if len(seatIDs) == 0 {
		return httpx.BadRequest("no_seats", "select at least one seat")
	}
	if len(seatIDs) > MaxSeatsPerBooking {
		return httpx.BadRequest("too_many_seats", "a booking may cover at most 8 seats")
	}
	seen := make(map[string]bool, len(seatIDs))
	for _, id := range seatIDs {
		if id == "" {
			return httpx.BadRequest("invalid_seat", "seat ids cannot be empty")
		}
		if seen[id] {
			return httpx.BadRequest("duplicate_seat", "the same seat was selected twice")
		}
		seen[id] = true
	}
	return nil
}

// Total sums the seat prices captured at hold time.
func Total(seats []Seat) int64 {
	var total int64
	for _, s := range seats {
		total += s.PriceCents
	}
	return total
}
