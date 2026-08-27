package domain

import (
	"strings"
	"testing"

	"github.com/ayush/delta-one/shared/events"
)

func TestMoney(t *testing.T) {
	if got := Money(430000); got != "₹4300.00" {
		t.Fatalf("got %q", got)
	}
	if got := Money(150050); got != "₹1500.50" {
		t.Fatalf("got %q", got)
	}
}

func TestTicketsConfirmed(t *testing.T) {
	msg := TicketsConfirmed(events.BookingConfirmed{
		BookingID:  "9f2c1b44-0000-4000-8000-000000000000",
		UserEmail:  "ada@example.com",
		EventTitle: "Harbour Jazz Sessions",
		SeatIDs:    []string{"a", "b"},
		TotalCents: 430000,
	})

	if msg.To != "ada@example.com" {
		t.Fatalf("to = %q", msg.To)
	}
	if !strings.Contains(msg.Body, "2 seats") {
		t.Fatalf("body does not say how many seats: %q", msg.Body)
	}
	if !strings.Contains(msg.Body, "9F2C1B44") {
		t.Fatalf("body has no readable reference: %q", msg.Body)
	}
}

func TestTicketsConfirmedSingularSeat(t *testing.T) {
	msg := TicketsConfirmed(events.BookingConfirmed{SeatIDs: []string{"a"}})
	if !strings.Contains(msg.Body, "1 seat for") {
		t.Fatalf("single seat rendered as %q", msg.Body)
	}
}
