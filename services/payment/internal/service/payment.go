// Package service holds the payment business logic.
package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/ayush/delta-one/services/payment/internal/domain"
	"github.com/ayush/delta-one/services/payment/internal/repository"
	"github.com/ayush/delta-one/shared/events"
	"github.com/google/uuid"
)

// Payments is the storage the service needs.
type Payments interface {
	Create(ctx context.Context, p domain.Payment) error
	SucceededByBooking(ctx context.Context, bookingID string) (domain.Payment, error)
	ByID(ctx context.Context, id string) (domain.Payment, error)
}

// Payment charges bookings through a simulated gateway.
type Payment struct {
	payments  Payments
	publisher *events.Publisher
}

// New returns a Payment service. publisher may be nil, in which case outcome
// events are not published.
func New(payments Payments, publisher *events.Publisher) *Payment {
	return &Payment{payments: payments, publisher: publisher}
}

// Charge attempts a payment for a booking.
//
// It is idempotent: charging a booking that already has a successful payment
// returns that payment instead of taking the money twice. The database unique
// index is the arbiter, so two concurrent charges cannot both succeed.
func (p *Payment) Charge(ctx context.Context, bookingID, userID, cardToken string, amountCents int64) (domain.Payment, error) {
	if err := domain.ValidateCharge(bookingID, userID, cardToken, amountCents); err != nil {
		return domain.Payment{}, err
	}

	if existing, err := p.payments.SucceededByBooking(ctx, bookingID); err == nil {
		return existing, nil
	} else if !errors.Is(err, domain.ErrPaymentNotFound) {
		return domain.Payment{}, err
	}

	record := domain.Payment{
		ID:          uuid.NewString(),
		BookingID:   bookingID,
		UserID:      userID,
		AmountCents: amountCents,
		Status:      domain.StatusSucceeded,
		CreatedAt:   time.Now().UTC(),
	}
	if reason := domain.Authorize(cardToken, amountCents); reason != "" {
		record.Status = domain.StatusFailed
		record.FailureReason = reason
	}

	if err := p.payments.Create(ctx, record); err != nil {
		// Another request charged this booking between the check above and
		// the insert. Its payment is the real one.
		if errors.Is(err, repository.ErrDuplicate) {
			return p.payments.SucceededByBooking(ctx, bookingID)
		}
		return domain.Payment{}, err
	}

	if record.Status == domain.StatusSucceeded {
		p.publish(ctx, events.TypePaymentSucceeded, events.PaymentSucceeded{
			PaymentID: record.ID, BookingID: bookingID, UserID: userID, AmountCents: amountCents,
		})
	} else {
		p.publish(ctx, events.TypePaymentFailed, events.PaymentFailed{
			BookingID: bookingID, UserID: userID, Reason: record.FailureReason,
		})
	}
	return record, nil
}

// Get returns one payment. A caller may only read their own.
func (p *Payment) Get(ctx context.Context, id, userID string) (domain.Payment, error) {
	payment, err := p.payments.ByID(ctx, id)
	if err != nil {
		return domain.Payment{}, err
	}
	if payment.UserID != userID {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	return payment, nil
}

func (p *Payment) publish(ctx context.Context, eventType string, payload any) {
	if p.publisher == nil {
		return
	}
	if err := p.publisher.Publish(ctx, eventType, payload); err != nil {
		log.Printf("payment: publish %s: %v", eventType, err)
	}
}
