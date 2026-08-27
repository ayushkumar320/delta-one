// Package repository stores catalog entities in Postgres.
package repository

import (
	"context"
	"fmt"

	"github.com/ayush/delta-one/services/catalog/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo reads and writes venues, events and seats.
type Repo struct {
	pool *pgxpool.Pool
}

// New returns a Repo backed by the given pool.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

// EventFilter narrows an event listing.
type EventFilter struct {
	City   string
	Search string
	Limit  int
	Offset int
}

// eventColumns joins each event to its venue and to its seat summary in one
// query, so a listing of N events costs one round trip rather than N+1.
const eventColumns = `
	SELECT e.id, e.venue_id, coalesce(e.organizer_id::text, ''), e.title, e.description,
	       e.starts_at, e.status, e.created_at,
	       v.id, v.name, v.city, v.address, v.created_at,
	       coalesce(s.seat_count, 0), coalesce(s.from_cents, 0)
	FROM events e
	JOIN venues v ON v.id = e.venue_id
	LEFT JOIN (
	    SELECT event_id, count(*) AS seat_count, min(price_cents) AS from_cents
	    FROM seats GROUP BY event_id
	) s ON s.event_id = e.id`

// ListEvents returns published events ordered by start time.
func (r *Repo) ListEvents(ctx context.Context, f EventFilter) ([]domain.Event, error) {
	if f.Limit <= 0 || f.Limit > domain.MaxPageSize {
		f.Limit = 20
	}
	rows, err := r.pool.Query(ctx, eventColumns+`
		WHERE e.status = 'published'
		  AND ($1 = '' OR v.city ILIKE $1)
		  AND ($2 = '' OR e.title ILIKE '%' || $2 || '%')
		ORDER BY e.starts_at
		LIMIT $3 OFFSET $4`, f.City, f.Search, f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	events := []domain.Event{}
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// EventByID returns one event with its venue and seat summary.
func (r *Repo) EventByID(ctx context.Context, id string) (domain.Event, error) {
	rows, err := r.pool.Query(ctx, eventColumns+` WHERE e.id = $1`, id)
	if err != nil {
		return domain.Event{}, fmt.Errorf("get event: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return domain.Event{}, fmt.Errorf("get event: %w", err)
		}
		return domain.Event{}, domain.ErrEventNotFound
	}
	return scanEvent(rows)
}

func scanEvent(rows pgx.Rows) (domain.Event, error) {
	var e domain.Event
	var v domain.Venue
	err := rows.Scan(&e.ID, &e.VenueID, &e.OrganizerID, &e.Title, &e.Description,
		&e.StartsAt, &e.Status, &e.CreatedAt,
		&v.ID, &v.Name, &v.City, &v.Address, &v.CreatedAt,
		&e.SeatCount, &e.FromCents)
	if err != nil {
		return domain.Event{}, fmt.Errorf("scan event: %w", err)
	}
	e.Venue = &v
	return e, nil
}

// SeatsByEvent returns every seat an event sells, in seat-map order.
func (r *Repo) SeatsByEvent(ctx context.Context, eventID string) ([]domain.Seat, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, event_id, section, row_label, seat_number, price_cents
		 FROM seats WHERE event_id = $1
		 ORDER BY section, row_label, seat_number`, eventID)
	if err != nil {
		return nil, fmt.Errorf("list seats: %w", err)
	}
	defer rows.Close()

	seats := []domain.Seat{}
	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.ID, &s.EventID, &s.Section, &s.RowLabel, &s.SeatNumber, &s.PriceCents); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		seats = append(seats, s)
	}
	return seats, rows.Err()
}

// SeatsByIDs returns the named seats of one event. The booking service uses it
// to price a hold, so seats belonging to another event are not returned.
func (r *Repo) SeatsByIDs(ctx context.Context, eventID string, ids []string) ([]domain.Seat, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, event_id, section, row_label, seat_number, price_cents
		 FROM seats WHERE event_id = $1 AND id = ANY($2)
		 ORDER BY section, row_label, seat_number`, eventID, ids)
	if err != nil {
		return nil, fmt.Errorf("list seats by id: %w", err)
	}
	defer rows.Close()

	seats := []domain.Seat{}
	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.ID, &s.EventID, &s.Section, &s.RowLabel, &s.SeatNumber, &s.PriceCents); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		seats = append(seats, s)
	}
	return seats, rows.Err()
}

// CreateVenue inserts a venue.
func (r *Repo) CreateVenue(ctx context.Context, v domain.Venue) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO venues (id, name, city, address) VALUES ($1, $2, $3, $4)`,
		v.ID, v.Name, v.City, v.Address)
	if err != nil {
		return fmt.Errorf("insert venue: %w", err)
	}
	return nil
}

// ListVenues returns every venue, ordered by city then name.
func (r *Repo) ListVenues(ctx context.Context) ([]domain.Venue, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, city, address, created_at FROM venues ORDER BY city, name`)
	if err != nil {
		return nil, fmt.Errorf("list venues: %w", err)
	}
	defer rows.Close()

	venues := []domain.Venue{}
	for rows.Next() {
		var v domain.Venue
		if err := rows.Scan(&v.ID, &v.Name, &v.City, &v.Address, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan venue: %w", err)
		}
		venues = append(venues, v)
	}
	return venues, rows.Err()
}

// CreateEvent inserts an event and its seats in one transaction, so an event
// never exists with a half-written seat map.
func (r *Repo) CreateEvent(ctx context.Context, e domain.Event, seats []domain.Seat) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var venueExists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM venues WHERE id = $1)`, e.VenueID).Scan(&venueExists); err != nil {
		return fmt.Errorf("check venue: %w", err)
	}
	if !venueExists {
		return domain.ErrVenueNotFound
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO events (id, venue_id, organizer_id, title, description, starts_at, status)
		 VALUES ($1, $2, nullif($3, '')::uuid, $4, $5, $6, $7)`,
		e.ID, e.VenueID, e.OrganizerID, e.Title, e.Description, e.StartsAt, e.Status)
	if err != nil {
		return fmt.Errorf("insert event: %w", err)
	}

	batch := &pgx.Batch{}
	for _, s := range seats {
		batch.Queue(
			`INSERT INTO seats (id, event_id, section, row_label, seat_number, price_cents)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			s.ID, e.ID, s.Section, s.RowLabel, s.SeatNumber, s.PriceCents)
	}
	if err := tx.SendBatch(ctx, batch).Close(); err != nil {
		return fmt.Errorf("insert seats: %w", err)
	}
	return tx.Commit(ctx)
}
