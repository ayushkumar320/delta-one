package domain

import (
	"testing"
	"time"
)

func TestValidateHold(t *testing.T) {
	if err := ValidateHold("event-1", []string{"a", "b"}); err != nil {
		t.Fatalf("valid hold rejected: %v", err)
	}
	if err := ValidateHold("event-1", []string{"a", "a"}); err == nil {
		t.Fatal("duplicate seat accepted")
	}
	if err := ValidateHold("event-1", nil); err == nil {
		t.Fatal("empty selection accepted")
	}
	many := make([]string, MaxSeatsPerBooking+1)
	for i := range many {
		many[i] = string(rune('a' + i))
	}
	if err := ValidateHold("event-1", many); err == nil {
		t.Fatal("oversized selection accepted")
	}
}

func TestExpiredOnlyAppliesToHeldBookings(t *testing.T) {
	past := time.Now().Add(-time.Minute)
	now := time.Now()

	held := Booking{Status: StatusHeld, HoldExpiresAt: past}
	if !held.Expired(now) {
		t.Fatal("a held booking past its window is not expired")
	}
	confirmed := Booking{Status: StatusConfirmed, HoldExpiresAt: past}
	if confirmed.Expired(now) {
		t.Fatal("a confirmed booking expired when its hold window passed")
	}
}

func TestTotal(t *testing.T) {
	if got := Total([]Seat{{PriceCents: 150000}, {PriceCents: 280000}}); got != 430000 {
		t.Fatalf("total = %d, want 430000", got)
	}
}
