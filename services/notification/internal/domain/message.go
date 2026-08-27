// Package domain turns platform events into the messages a customer receives.
// It imports no other layer of the service.
package domain

import (
	"fmt"
	"strings"

	"github.com/ayush/delta-one/shared/events"
)

// Message is one rendered notification, ready to hand to an email or SMS
// provider.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Money formats cents as rupees for display.
func Money(cents int64) string {
	return fmt.Sprintf("₹%d.%02d", cents/100, cents%100)
}

// Welcome greets a new account.
func Welcome(e events.UserRegistered) Message {
	return Message{
		To:      e.Email,
		Subject: "Welcome to Delta One",
		Body: fmt.Sprintf("Hi %s, your account is ready. Browse what is on sale and "+
			"your tickets will appear under My bookings.", e.Name),
	}
}

// TicketsConfirmed confirms a paid booking.
func TicketsConfirmed(e events.BookingConfirmed) Message {
	seats := "1 seat"
	if len(e.SeatIDs) != 1 {
		seats = fmt.Sprintf("%d seats", len(e.SeatIDs))
	}
	return Message{
		To:      e.UserEmail,
		Subject: "Your tickets for " + e.EventTitle,
		Body: fmt.Sprintf("Confirmed: %s for %s, %s. Booking reference %s.",
			seats, e.EventTitle, Money(e.TotalCents), reference(e.BookingID)),
	}
}

// BookingReleased tells a customer their seats went back on sale.
func BookingReleased(e events.BookingCancelled) Message {
	return Message{
		To:      e.UserEmail,
		Subject: "Your booking was cancelled",
		Body: fmt.Sprintf("Booking %s was cancelled: %s. No payment was taken.",
			reference(e.BookingID), e.Reason),
	}
}

// reference shortens a booking UUID into something a person can read out.
func reference(bookingID string) string {
	head, _, _ := strings.Cut(bookingID, "-")
	return strings.ToUpper(head)
}
