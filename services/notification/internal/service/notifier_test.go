package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ayush/delta-one/services/notification/internal/domain"
	"github.com/ayush/delta-one/shared/events"
)

type recorder struct {
	sent []domain.Message
	err  error
}

func (r *recorder) Send(_ context.Context, msg domain.Message) error {
	if r.err != nil {
		return r.err
	}
	r.sent = append(r.sent, msg)
	return nil
}

func envelope(t *testing.T, eventType string, payload any) events.Envelope {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return events.Envelope{Type: eventType, Data: data}
}

func TestHandleSendsForKnownEvents(t *testing.T) {
	rec := &recorder{}
	n := New(rec)
	ctx := context.Background()

	err := n.Handle(ctx, envelope(t, events.TypeBookingConfirmed, events.BookingConfirmed{
		UserEmail: "ada@example.com", EventTitle: "Harbour Jazz Sessions", SeatIDs: []string{"a"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.sent) != 1 || rec.sent[0].To != "ada@example.com" {
		t.Fatalf("sent = %+v", rec.sent)
	}
}

func TestHandleIgnoresUnknownEvents(t *testing.T) {
	rec := &recorder{}
	if err := New(rec).Handle(context.Background(), envelope(t, "seat.repriced", map[string]string{})); err != nil {
		t.Fatalf("unknown event returned an error, so it would be retried forever: %v", err)
	}
	if len(rec.sent) != 0 {
		t.Fatal("unknown event produced a message")
	}
}

func TestHandleReturnsSenderFailuresSoTheEventIsRetried(t *testing.T) {
	rec := &recorder{err: errors.New("smtp unavailable")}
	err := New(rec).Handle(context.Background(), envelope(t, events.TypeUserRegistered,
		events.UserRegistered{Email: "ada@example.com", Name: "Ada"}))
	if err == nil {
		t.Fatal("a failed delivery was acknowledged")
	}
}

func TestHandleDropsEventsWithNoRecipient(t *testing.T) {
	rec := &recorder{}
	err := New(rec).Handle(context.Background(), envelope(t, events.TypeBookingCancelled,
		events.BookingCancelled{BookingID: "b1", Reason: "expired"}))
	if err != nil {
		t.Fatalf("event with no recipient was retried: %v", err)
	}
	if len(rec.sent) != 0 {
		t.Fatal("message sent with no recipient")
	}
}
