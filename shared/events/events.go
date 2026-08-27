// Package events defines the platform's asynchronous event payloads and the
// Redis Streams publisher and consumer that carry them.
//
// One stream holds every event. Consumers join a consumer group named after
// their service, so each service sees every event exactly once and a restarted
// consumer resumes where it left off.
package events

import (
	"encoding/json"
	"time"
)

// Stream is the Redis Streams key every event is written to.
const Stream = "delta-one.events"

// Event types published by the platform.
const (
	TypeUserRegistered   = "user.registered"
	TypeBookingHeld      = "booking.held"
	TypeBookingConfirmed = "booking.confirmed"
	TypeBookingCancelled = "booking.cancelled"
	TypePaymentSucceeded = "payment.succeeded"
	TypePaymentFailed    = "payment.failed"
)

// Envelope wraps every published payload with the metadata a consumer needs to
// route, deduplicate and trace it.
type Envelope struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	OccurredAt time.Time       `json:"occurred_at"`
	RequestID  string          `json:"request_id,omitempty"`
	Data       json.RawMessage `json:"data"`
}

// Into decodes the envelope payload into v.
func (e Envelope) Into(v any) error { return json.Unmarshal(e.Data, v) }

// UserRegistered is published when a new account is created.
type UserRegistered struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// BookingConfirmed is published once a booking's payment has succeeded.
type BookingConfirmed struct {
	BookingID  string   `json:"booking_id"`
	UserID     string   `json:"user_id"`
	UserEmail  string   `json:"user_email"`
	EventID    string   `json:"event_id"`
	EventTitle string   `json:"event_title"`
	SeatIDs    []string `json:"seat_ids"`
	TotalCents int64    `json:"total_cents"`
}

// BookingCancelled is published when a booking is cancelled or its hold expires.
type BookingCancelled struct {
	BookingID string `json:"booking_id"`
	UserID    string `json:"user_id"`
	UserEmail string `json:"user_email"`
	Reason    string `json:"reason"`
}

// PaymentSucceeded and PaymentFailed report the outcome of a charge.
type PaymentSucceeded struct {
	PaymentID   string `json:"payment_id"`
	BookingID   string `json:"booking_id"`
	UserID      string `json:"user_id"`
	AmountCents int64  `json:"amount_cents"`
}

type PaymentFailed struct {
	BookingID string `json:"booking_id"`
	UserID    string `json:"user_id"`
	Reason    string `json:"reason"`
}
