package service

import (
	"context"
	"testing"

	"github.com/ayush/delta-one/services/payment/internal/domain"
	"github.com/ayush/delta-one/services/payment/internal/repository"
)

// memPayments enforces the same "one succeeded payment per booking" rule as
// the database's partial unique index.
type memPayments struct {
	byID map[string]domain.Payment
}

func newMem() *memPayments { return &memPayments{byID: map[string]domain.Payment{}} }

func (m *memPayments) Create(_ context.Context, p domain.Payment) error {
	if p.Status == domain.StatusSucceeded {
		for _, existing := range m.byID {
			if existing.BookingID == p.BookingID && existing.Status == domain.StatusSucceeded {
				return repository.ErrDuplicate
			}
		}
	}
	m.byID[p.ID] = p
	return nil
}

func (m *memPayments) SucceededByBooking(_ context.Context, bookingID string) (domain.Payment, error) {
	for _, p := range m.byID {
		if p.BookingID == bookingID && p.Status == domain.StatusSucceeded {
			return p, nil
		}
	}
	return domain.Payment{}, domain.ErrPaymentNotFound
}

func (m *memPayments) ByID(_ context.Context, id string) (domain.Payment, error) {
	p, ok := m.byID[id]
	if !ok {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return p, nil
}

func TestChargeIsIdempotent(t *testing.T) {
	svc := New(newMem(), nil)
	ctx := context.Background()

	first, err := svc.Charge(ctx, "booking-1", "user-1", "tok_visa", 250000)
	if err != nil {
		t.Fatal(err)
	}
	if first.Status != domain.StatusSucceeded {
		t.Fatalf("status = %q, want succeeded", first.Status)
	}

	second, err := svc.Charge(ctx, "booking-1", "user-1", "tok_visa", 250000)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID != first.ID {
		t.Fatalf("second charge created payment %s, want the original %s", second.ID, first.ID)
	}
}

func TestChargeRecordsDeclines(t *testing.T) {
	svc := New(newMem(), nil)
	got, err := svc.Charge(context.Background(), "booking-2", "user-1", domain.TokenDecline, 250000)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != domain.StatusFailed || got.FailureReason == "" {
		t.Fatalf("declined charge recorded as %+v", got)
	}
}

func TestGetRefusesAnotherUsersPayment(t *testing.T) {
	svc := New(newMem(), nil)
	ctx := context.Background()
	paid, _ := svc.Charge(ctx, "booking-3", "user-1", "tok_visa", 250000)

	if _, err := svc.Get(ctx, paid.ID, "user-2"); err != domain.ErrPaymentNotFound {
		t.Fatalf("err = %v, want ErrPaymentNotFound", err)
	}
}
