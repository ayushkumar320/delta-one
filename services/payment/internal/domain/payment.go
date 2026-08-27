// Package domain holds the payment entities and the rules of the simulated
// gateway. It imports no other layer of the service.
package domain

import (
	"strings"
	"time"

	"github.com/ayush/delta-one/shared/httpx"
)

// Payment outcomes.
const (
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// Payment is one attempt to charge a booking.
type Payment struct {
	ID            string    `json:"id"`
	BookingID     string    `json:"booking_id"`
	UserID        string    `json:"user_id"`
	AmountCents   int64     `json:"amount_cents"`
	Status        string    `json:"status"`
	FailureReason string    `json:"failure_reason,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// Card tokens the simulated gateway treats specially. Real gateways return
// these outcomes from the network; here they are chosen by the token so the
// failure paths can be exercised without one.
const (
	TokenDecline      = "tok_decline"
	TokenInsufficient = "tok_insufficient_funds"
)

// Errors the payment service returns.
var (
	ErrPaymentNotFound = httpx.NotFound("payment_not_found", "payment not found")
	ErrAlreadyCharged  = httpx.Conflict("already_charged", "this booking has already been paid for")
)

// ValidateCharge checks a charge request before the gateway is called.
func ValidateCharge(bookingID, userID, cardToken string, amountCents int64) error {
	if bookingID == "" || userID == "" {
		return httpx.BadRequest("invalid_charge", "booking_id and user_id are required")
	}
	if amountCents <= 0 {
		return httpx.BadRequest("invalid_amount", "amount_cents must be greater than zero")
	}
	if !strings.HasPrefix(cardToken, "tok_") {
		return httpx.BadRequest("invalid_card", "card_token must be a tokenised card")
	}
	return nil
}

// Authorize is the simulated gateway decision. It returns an empty reason when
// the charge succeeds.
func Authorize(cardToken string, amountCents int64) (failureReason string) {
	switch cardToken {
	case TokenDecline:
		return "the card was declined"
	case TokenInsufficient:
		return "insufficient funds"
	}
	// A real gateway enforces per-transaction limits; this stands in for one.
	if amountCents > 5_000_000 {
		return "amount exceeds the per-transaction limit"
	}
	return ""
}
