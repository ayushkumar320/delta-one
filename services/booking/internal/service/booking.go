// Package service holds the booking business logic: taking a seat hold,
// confirming it with a payment, and releasing it when it is abandoned.
package service

import (
	"context"
	"log"
	"time"

	"github.com/ayush/delta-one/services/booking/internal/client"
	"github.com/ayush/delta-one/services/booking/internal/domain"
	"github.com/ayush/delta-one/shared/events"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/google/uuid"
)

// Store is the storage the booking service needs.
type Store interface {
	Hold(ctx context.Context, b domain.Booking) error
	ByID(ctx context.Context, id string) (domain.Booking, error)
	ListByUser(ctx context.Context, userID string) ([]domain.Booking, error)
	Confirm(ctx context.Context, id string) error
	Release(ctx context.Context, id, status string) error
	TakenSeats(ctx context.Context, eventID string) ([]string, error)
}

// CatalogClient prices seats and names events.
type CatalogClient interface {
	Event(ctx context.Context, eventID string) (client.Event, error)
	Seats(ctx context.Context, eventID string, seatIDs []string) ([]client.Seat, error)
}

// PaymentClient charges a booking.
type PaymentClient interface {
	Charge(ctx context.Context, bookingID, cardToken string, amountCents int64) (client.Charge, error)
}

// Booking coordinates holds, payments and cancellations.
type Booking struct {
	store     Store
	catalog   CatalogClient
	payment   PaymentClient
	publisher *events.Publisher
}

// New returns a Booking service. publisher may be nil, in which case events
// are not published.
func New(store Store, catalog CatalogClient, payment PaymentClient, publisher *events.Publisher) *Booking {
	return &Booking{store: store, catalog: catalog, payment: payment, publisher: publisher}
}

// Hold reserves seats for a user for domain.HoldTTL.
//
// Prices come from the catalog rather than the client, and are frozen onto the
// booking, so the user pays what they were shown even if the event is
// repriced mid-checkout.
func (b *Booking) Hold(ctx context.Context, userID, userEmail, eventID string, seatIDs []string) (domain.Booking, error) {
	if err := domain.ValidateHold(eventID, seatIDs); err != nil {
		return domain.Booking{}, err
	}

	event, err := b.catalog.Event(ctx, eventID)
	if err != nil {
		return domain.Booking{}, err
	}
	if event.Status != "published" {
		return domain.Booking{}, httpx.Conflict("event_unavailable", "this event is not on sale")
	}

	seats, err := b.catalog.Seats(ctx, eventID, seatIDs)
	if err != nil {
		return domain.Booking{}, err
	}

	held := make([]domain.Seat, len(seats))
	for i, s := range seats {
		held[i] = domain.Seat{SeatID: s.ID, PriceCents: s.PriceCents}
	}

	now := time.Now().UTC()
	booking := domain.Booking{
		ID:            uuid.NewString(),
		UserID:        userID,
		UserEmail:     userEmail,
		EventID:       eventID,
		EventTitle:    event.Title,
		Status:        domain.StatusHeld,
		TotalCents:    domain.Total(held),
		HoldExpiresAt: now.Add(domain.HoldTTL),
		CreatedAt:     now,
		UpdatedAt:     now,
		Seats:         held,
	}
	if err := b.store.Hold(ctx, booking); err != nil {
		return domain.Booking{}, err
	}
	return booking, nil
}

// Confirm charges the held seats and confirms the booking.
//
// The charge runs before the status change, and payment is idempotent, so a
// retry after a failure here re-uses the original payment rather than charging
// the card twice.
func (b *Booking) Confirm(ctx context.Context, bookingID, userID, cardToken string) (domain.Booking, error) {
	booking, err := b.owned(ctx, bookingID, userID)
	if err != nil {
		return domain.Booking{}, err
	}
	switch {
	case booking.Status == domain.StatusConfirmed:
		return domain.Booking{}, domain.ErrAlreadyConfirmed
	case booking.Expired(time.Now()):
		return domain.Booking{}, domain.ErrHoldExpired
	case booking.Status != domain.StatusHeld:
		return domain.Booking{}, domain.ErrNotHeld
	}

	charge, err := b.payment.Charge(ctx, booking.ID, cardToken, booking.TotalCents)
	if err != nil {
		return domain.Booking{}, err
	}
	if !charge.Succeeded() {
		// The hold survives a decline, so the user can retry with another
		// card instead of losing their seats to the next buyer.
		return domain.Booking{}, httpx.NewError(402, "payment_failed", charge.FailureReason)
	}

	if err := b.store.Confirm(ctx, booking.ID); err != nil {
		return domain.Booking{}, err
	}
	booking.Status = domain.StatusConfirmed
	booking.UpdatedAt = time.Now().UTC()

	seatIDs := make([]string, len(booking.Seats))
	for i, s := range booking.Seats {
		seatIDs[i] = s.SeatID
	}
	b.publish(ctx, events.TypeBookingConfirmed, events.BookingConfirmed{
		BookingID:  booking.ID,
		UserID:     booking.UserID,
		UserEmail:  booking.UserEmail,
		EventID:    booking.EventID,
		EventTitle: booking.EventTitle,
		SeatIDs:    seatIDs,
		TotalCents: booking.TotalCents,
	})
	return booking, nil
}

// Cancel releases a booking's seats.
func (b *Booking) Cancel(ctx context.Context, bookingID, userID string) error {
	booking, err := b.owned(ctx, bookingID, userID)
	if err != nil {
		return err
	}
	if err := b.store.Release(ctx, booking.ID, domain.StatusCancelled); err != nil {
		return err
	}
	b.publish(ctx, events.TypeBookingCancelled, events.BookingCancelled{
		BookingID: booking.ID,
		UserID:    booking.UserID,
		UserEmail: booking.UserEmail,
		Reason:    "cancelled by the customer",
	})
	return nil
}

// Get returns one of the user's bookings.
func (b *Booking) Get(ctx context.Context, bookingID, userID string) (domain.Booking, error) {
	return b.owned(ctx, bookingID, userID)
}

// List returns the user's bookings, newest first.
func (b *Booking) List(ctx context.Context, userID string) ([]domain.Booking, error) {
	return b.store.ListByUser(ctx, userID)
}

// TakenSeats returns the seats of an event that are currently claimed.
func (b *Booking) TakenSeats(ctx context.Context, eventID string) ([]string, error) {
	return b.store.TakenSeats(ctx, eventID)
}

// owned loads a booking and hides it from anyone but its owner. Answering
// "not found" rather than "forbidden" keeps booking IDs from being probed.
func (b *Booking) owned(ctx context.Context, bookingID, userID string) (domain.Booking, error) {
	booking, err := b.store.ByID(ctx, bookingID)
	if err != nil {
		return domain.Booking{}, err
	}
	if booking.UserID != userID {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	return booking, nil
}

func (b *Booking) publish(ctx context.Context, eventType string, payload any) {
	if b.publisher == nil {
		return
	}
	if err := b.publisher.Publish(ctx, eventType, payload); err != nil {
		log.Printf("booking: publish %s: %v", eventType, err)
	}
}
