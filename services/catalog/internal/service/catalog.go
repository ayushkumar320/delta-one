// Package service holds the catalog business logic.
package service

import (
	"context"
	"strings"
	"time"

	"github.com/ayush/delta-one/services/catalog/internal/domain"
	"github.com/ayush/delta-one/services/catalog/internal/repository"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/google/uuid"
)

// Store is the storage the catalog needs.
type Store interface {
	ListEvents(ctx context.Context, f repository.EventFilter) ([]domain.Event, error)
	EventByID(ctx context.Context, id string) (domain.Event, error)
	SeatsByEvent(ctx context.Context, eventID string) ([]domain.Seat, error)
	SeatsByIDs(ctx context.Context, eventID string, ids []string) ([]domain.Seat, error)
	ListVenues(ctx context.Context) ([]domain.Venue, error)
	CreateVenue(ctx context.Context, v domain.Venue) error
	CreateEvent(ctx context.Context, e domain.Event, seats []domain.Seat) error
}

// Catalog answers questions about what is on sale.
type Catalog struct {
	store Store
}

// New returns a Catalog backed by store.
func New(store Store) *Catalog { return &Catalog{store: store} }

// ListEvents returns published events matching the filter.
func (c *Catalog) ListEvents(ctx context.Context, f repository.EventFilter) ([]domain.Event, error) {
	f.City = strings.TrimSpace(f.City)
	f.Search = strings.TrimSpace(f.Search)
	if f.Offset < 0 {
		f.Offset = 0
	}
	return c.store.ListEvents(ctx, f)
}

// Event returns one event. Drafts and cancelled events are visible only to
// their organizer, so a guessed ID does not expose an unpublished lineup.
func (c *Catalog) Event(ctx context.Context, id, viewerID string) (domain.Event, error) {
	event, err := c.store.EventByID(ctx, id)
	if err != nil {
		return domain.Event{}, err
	}
	if event.Status != domain.StatusPublished && event.OrganizerID != viewerID {
		return domain.Event{}, domain.ErrEventNotFound
	}
	return event, nil
}

// Seats returns an event's seat map.
func (c *Catalog) Seats(ctx context.Context, eventID, viewerID string) ([]domain.Seat, error) {
	if _, err := c.Event(ctx, eventID, viewerID); err != nil {
		return nil, err
	}
	return c.store.SeatsByEvent(ctx, eventID)
}

// SeatsByIDs returns the named seats of an event. It reports a missing seat as
// a bad request rather than silently pricing a shorter list, so the booking
// service cannot hold seats it did not ask for.
func (c *Catalog) SeatsByIDs(ctx context.Context, eventID string, ids []string) ([]domain.Seat, error) {
	if len(ids) == 0 {
		return nil, httpx.BadRequest("no_seats", "at least one seat id is required")
	}
	seats, err := c.store.SeatsByIDs(ctx, eventID, ids)
	if err != nil {
		return nil, err
	}
	if len(seats) != len(ids) {
		return nil, httpx.BadRequest("unknown_seat", "one or more seats do not belong to this event")
	}
	return seats, nil
}

// Venues returns every venue.
func (c *Catalog) Venues(ctx context.Context) ([]domain.Venue, error) {
	return c.store.ListVenues(ctx)
}

// CreateVenue adds a venue.
func (c *Catalog) CreateVenue(ctx context.Context, name, city, address string) (domain.Venue, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(city) == "" {
		return domain.Venue{}, httpx.BadRequest("invalid_venue", "name and city are required")
	}
	venue := domain.Venue{
		ID: uuid.NewString(), Name: name, City: city, Address: address,
		CreatedAt: time.Now().UTC(),
	}
	if err := c.store.CreateVenue(ctx, venue); err != nil {
		return domain.Venue{}, err
	}
	return venue, nil
}

// NewEvent is an event and its seat map as submitted by an organizer.
type NewEvent struct {
	VenueID     string
	Title       string
	Description string
	StartsAt    time.Time
	Status      string
	Seats       []domain.Seat
}

// CreateEvent stores an event and the seats it sells.
func (c *Catalog) CreateEvent(ctx context.Context, organizerID string, in NewEvent) (domain.Event, error) {
	if in.Status == "" {
		in.Status = domain.StatusPublished
	}
	if err := domain.ValidateEvent(in.Title, in.VenueID, in.StartsAt, in.Status); err != nil {
		return domain.Event{}, err
	}
	if err := domain.ValidateSeats(in.Seats); err != nil {
		return domain.Event{}, err
	}

	event := domain.Event{
		ID:          uuid.NewString(),
		VenueID:     in.VenueID,
		OrganizerID: organizerID,
		Title:       in.Title,
		Description: in.Description,
		StartsAt:    in.StartsAt,
		Status:      in.Status,
		CreatedAt:   time.Now().UTC(),
	}
	seats := make([]domain.Seat, len(in.Seats))
	for i, s := range in.Seats {
		s.ID = uuid.NewString()
		s.EventID = event.ID
		seats[i] = s
	}
	if err := c.store.CreateEvent(ctx, event, seats); err != nil {
		return domain.Event{}, err
	}
	event.SeatCount = len(seats)
	return event, nil
}
