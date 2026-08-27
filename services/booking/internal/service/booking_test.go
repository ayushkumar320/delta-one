package service

import (
	"context"
	"testing"
	"time"

	"github.com/ayush/delta-one/services/booking/internal/client"
	"github.com/ayush/delta-one/services/booking/internal/domain"
)

// memStore mirrors the database's guarantee: a seat can belong to at most one
// unreleased booking.
type memStore struct {
	bookings map[string]domain.Booking
	claimed  map[string]string // seat id -> booking id
}

func newStore() *memStore {
	return &memStore{bookings: map[string]domain.Booking{}, claimed: map[string]string{}}
}

func (m *memStore) Hold(_ context.Context, b domain.Booking) error {
	for _, s := range b.Seats {
		if _, taken := m.claimed[s.SeatID]; taken {
			return domain.ErrSeatsTaken
		}
	}
	for _, s := range b.Seats {
		m.claimed[s.SeatID] = b.ID
	}
	m.bookings[b.ID] = b
	return nil
}

func (m *memStore) ByID(_ context.Context, id string) (domain.Booking, error) {
	b, ok := m.bookings[id]
	if !ok {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	return b, nil
}

func (m *memStore) ListByUser(_ context.Context, userID string) ([]domain.Booking, error) {
	out := []domain.Booking{}
	for _, b := range m.bookings {
		if b.UserID == userID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (m *memStore) Confirm(_ context.Context, id string) error {
	b := m.bookings[id]
	if b.Status != domain.StatusHeld || time.Now().After(b.HoldExpiresAt) {
		return domain.ErrHoldExpired
	}
	b.Status = domain.StatusConfirmed
	m.bookings[id] = b
	return nil
}

func (m *memStore) Release(_ context.Context, id, status string) error {
	b, ok := m.bookings[id]
	if !ok {
		return domain.ErrBookingNotFound
	}
	b.Status = status
	m.bookings[id] = b
	for _, s := range b.Seats {
		delete(m.claimed, s.SeatID)
	}
	return nil
}

func (m *memStore) TakenSeats(context.Context, string) ([]string, error) {
	seats := []string{}
	for seatID := range m.claimed {
		seats = append(seats, seatID)
	}
	return seats, nil
}

type fakeCatalog struct{ status string }

func (f fakeCatalog) Event(context.Context, string) (client.Event, error) {
	status := f.status
	if status == "" {
		status = "published"
	}
	return client.Event{ID: "event-1", Title: "Midnight Synth Orchestra", Status: status}, nil
}

func (fakeCatalog) Seats(_ context.Context, _ string, ids []string) ([]client.Seat, error) {
	seats := make([]client.Seat, len(ids))
	for i, id := range ids {
		seats[i] = client.Seat{ID: id, PriceCents: 150000}
	}
	return seats, nil
}

type fakePayment struct {
	calls   int
	succeed bool
}

func (f *fakePayment) Charge(_ context.Context, _, _ string, amount int64) (client.Charge, error) {
	f.calls++
	if !f.succeed {
		return client.Charge{Status: "failed", FailureReason: "the card was declined"}, nil
	}
	return client.Charge{ID: "pay-1", Status: "succeeded", AmountCents: amount}, nil
}

func newBooking(store Store, pay PaymentClient) *Booking {
	return New(store, fakeCatalog{}, pay, nil)
}

func TestHoldPricesFromTheCatalog(t *testing.T) {
	svc := newBooking(newStore(), &fakePayment{succeed: true})
	held, err := svc.Hold(context.Background(), "user-1", "a@example.com", "event-1", []string{"seat-1", "seat-2"})
	if err != nil {
		t.Fatal(err)
	}
	if held.TotalCents != 300000 {
		t.Fatalf("total = %d, want 300000", held.TotalCents)
	}
	if held.Status != domain.StatusHeld {
		t.Fatalf("status = %q, want held", held.Status)
	}
	if held.HoldExpiresAt.Before(time.Now()) {
		t.Fatal("hold expires in the past")
	}
}

func TestHoldRefusesSeatsAlreadyClaimed(t *testing.T) {
	store := newStore()
	svc := newBooking(store, &fakePayment{succeed: true})
	ctx := context.Background()

	if _, err := svc.Hold(ctx, "user-1", "a@example.com", "event-1", []string{"seat-1"}); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Hold(ctx, "user-2", "b@example.com", "event-1", []string{"seat-1"})
	if err != domain.ErrSeatsTaken {
		t.Fatalf("err = %v, want ErrSeatsTaken", err)
	}
}

func TestConfirmKeepsTheHoldWhenPaymentIsDeclined(t *testing.T) {
	store := newStore()
	pay := &fakePayment{succeed: false}
	svc := newBooking(store, pay)
	ctx := context.Background()

	held, _ := svc.Hold(ctx, "user-1", "a@example.com", "event-1", []string{"seat-1"})
	if _, err := svc.Confirm(ctx, held.ID, "user-1", "tok_decline"); err == nil {
		t.Fatal("declined payment confirmed the booking")
	}

	after, _ := store.ByID(ctx, held.ID)
	if after.Status != domain.StatusHeld {
		t.Fatalf("status after decline = %q, want held so the user can retry", after.Status)
	}

	// Retrying with a card that works must still succeed on the same hold.
	pay.succeed = true
	confirmed, err := svc.Confirm(ctx, held.ID, "user-1", "tok_visa")
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Status != domain.StatusConfirmed {
		t.Fatalf("status = %q, want confirmed", confirmed.Status)
	}
}

func TestConfirmRefusesAnExpiredHold(t *testing.T) {
	store := newStore()
	pay := &fakePayment{succeed: true}
	svc := newBooking(store, pay)
	ctx := context.Background()

	held, _ := svc.Hold(ctx, "user-1", "a@example.com", "event-1", []string{"seat-1"})
	stale := store.bookings[held.ID]
	stale.HoldExpiresAt = time.Now().Add(-time.Minute)
	store.bookings[held.ID] = stale

	if _, err := svc.Confirm(ctx, held.ID, "user-1", "tok_visa"); err != domain.ErrHoldExpired {
		t.Fatalf("err = %v, want ErrHoldExpired", err)
	}
	if pay.calls != 0 {
		t.Fatal("an expired hold was charged")
	}
}

func TestConfirmRefusesAnotherUsersBooking(t *testing.T) {
	svc := newBooking(newStore(), &fakePayment{succeed: true})
	ctx := context.Background()

	held, _ := svc.Hold(ctx, "user-1", "a@example.com", "event-1", []string{"seat-1"})
	if _, err := svc.Confirm(ctx, held.ID, "user-2", "tok_visa"); err != domain.ErrBookingNotFound {
		t.Fatalf("err = %v, want ErrBookingNotFound", err)
	}
}

func TestCancelReturnsSeatsToThePool(t *testing.T) {
	store := newStore()
	svc := newBooking(store, &fakePayment{succeed: true})
	ctx := context.Background()

	held, _ := svc.Hold(ctx, "user-1", "a@example.com", "event-1", []string{"seat-1"})
	if err := svc.Cancel(ctx, held.ID, "user-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Hold(ctx, "user-2", "b@example.com", "event-1", []string{"seat-1"}); err != nil {
		t.Fatalf("seat was not released: %v", err)
	}
}
