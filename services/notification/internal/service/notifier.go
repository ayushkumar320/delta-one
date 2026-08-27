// Package service consumes platform events and delivers the notifications
// they call for.
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/ayush/delta-one/services/notification/internal/domain"
	"github.com/ayush/delta-one/shared/events"
)

// Sender delivers a rendered message. Swapping the log sender for a real email
// or SMS provider is the only change this service needs to become real.
type Sender interface {
	Send(ctx context.Context, msg domain.Message) error
}

// LogSender writes messages to the log instead of sending them.
type LogSender struct{}

// Send prints the message.
func (LogSender) Send(_ context.Context, msg domain.Message) error {
	log.Printf("notification: to=%s subject=%q body=%q", msg.To, msg.Subject, msg.Body)
	return nil
}

// Notifier turns events into messages.
type Notifier struct {
	sender Sender
}

// New returns a Notifier that delivers through sender.
func New(sender Sender) *Notifier { return &Notifier{sender: sender} }

// Handle renders one event and delivers it. Returning an error leaves the
// event unacknowledged so it is retried; unknown event types are not an error,
// because a service publishing something this one does not care about is
// normal.
func (n *Notifier) Handle(ctx context.Context, env events.Envelope) error {
	var msg domain.Message

	switch env.Type {
	case events.TypeUserRegistered:
		var e events.UserRegistered
		if err := env.Into(&e); err != nil {
			return fmt.Errorf("decode %s: %w", env.Type, err)
		}
		msg = domain.Welcome(e)

	case events.TypeBookingConfirmed:
		var e events.BookingConfirmed
		if err := env.Into(&e); err != nil {
			return fmt.Errorf("decode %s: %w", env.Type, err)
		}
		msg = domain.TicketsConfirmed(e)

	case events.TypeBookingCancelled:
		var e events.BookingCancelled
		if err := env.Into(&e); err != nil {
			return fmt.Errorf("decode %s: %w", env.Type, err)
		}
		msg = domain.BookingReleased(e)

	default:
		return nil
	}

	if msg.To == "" {
		// Nothing to deliver to. Acknowledging is right: retrying will not
		// conjure an address.
		log.Printf("notification: %s carried no recipient, dropping", env.Type)
		return nil
	}
	return n.sender.Send(ctx, msg)
}
