// Package repository stores bookings in Postgres.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ayush/delta-one/services/booking/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo reads and writes bookings and the seats they claim.
type Repo struct {
	pool *pgxpool.Pool
}

// New returns a Repo backed by the given pool.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const uniqueViolation = "23505"

// releaseExpired returns the seats of holds whose window has passed. It runs
// inside the hold transaction so correctness does not depend on the background
// sweeper having run recently: an abandoned hold stops blocking a seat the
// moment someone else tries to take it.
const releaseExpired = `
	WITH stale AS (
	    UPDATE bookings SET status = 'expired', updated_at = now()
	    WHERE event_id = $1 AND status = 'held' AND hold_expires_at < now()
	    RETURNING id
	)
	UPDATE booking_seats SET released_at = now()
	WHERE booking_id IN (SELECT id FROM stale) AND released_at IS NULL`

// Hold writes a booking and its seat claims in one transaction. The partial
// unique index on unreleased booking_seats rows is what makes a double booking
// impossible: two concurrent requests for the same seat both reach the insert,
// and exactly one of them commits.
func (r *Repo) Hold(ctx context.Context, b domain.Booking) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin hold: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, releaseExpired, b.EventID); err != nil {
		return fmt.Errorf("release expired holds: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO bookings
		 (id, user_id, user_email, event_id, event_title, status, total_cents, hold_expires_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		b.ID, b.UserID, b.UserEmail, b.EventID, b.EventTitle, b.Status, b.TotalCents, b.HoldExpiresAt)
	if err != nil {
		return fmt.Errorf("insert booking: %w", err)
	}

	for _, s := range b.Seats {
		_, err := tx.Exec(ctx,
			`INSERT INTO booking_seats (booking_id, event_id, seat_id, price_cents)
			 VALUES ($1, $2, $3, $4)`,
			b.ID, b.EventID, s.SeatID, s.PriceCents)
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return domain.ErrSeatsTaken
		}
		if err != nil {
			return fmt.Errorf("insert booking seat: %w", err)
		}
	}
	return tx.Commit(ctx)
}

const selectBooking = `
	SELECT id, user_id, user_email, event_id, event_title, status, total_cents,
	       hold_expires_at, created_at, updated_at
	FROM bookings`

// ByID returns one booking with its seats.
func (r *Repo) ByID(ctx context.Context, id string) (domain.Booking, error) {
	var b domain.Booking
	err := r.pool.QueryRow(ctx, selectBooking+` WHERE id = $1`, id).Scan(
		&b.ID, &b.UserID, &b.UserEmail, &b.EventID, &b.EventTitle, &b.Status,
		&b.TotalCents, &b.HoldExpiresAt, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Booking{}, domain.ErrBookingNotFound
	}
	if err != nil {
		return domain.Booking{}, fmt.Errorf("select booking: %w", err)
	}
	b.Seats, err = r.seats(ctx, b.ID)
	if err != nil {
		return domain.Booking{}, err
	}
	return b, nil
}

// ListByUser returns a user's bookings, newest first, each with its seats.
func (r *Repo) ListByUser(ctx context.Context, userID string) ([]domain.Booking, error) {
	rows, err := r.pool.Query(ctx, selectBooking+`
		WHERE user_id = $1 ORDER BY created_at DESC LIMIT 100`, userID)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	bookings := []domain.Booking{}
	ids := []string{}
	for rows.Next() {
		var b domain.Booking
		if err := rows.Scan(&b.ID, &b.UserID, &b.UserEmail, &b.EventID, &b.EventTitle,
			&b.Status, &b.TotalCents, &b.HoldExpiresAt, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		bookings = append(bookings, b)
		ids = append(ids, b.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(bookings) == 0 {
		return bookings, nil
	}

	// One query for every booking's seats rather than one per booking.
	seatRows, err := r.pool.Query(ctx,
		`SELECT booking_id, seat_id, price_cents FROM booking_seats WHERE booking_id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("list booking seats: %w", err)
	}
	defer seatRows.Close()

	byBooking := map[string][]domain.Seat{}
	for seatRows.Next() {
		var bookingID string
		var s domain.Seat
		if err := seatRows.Scan(&bookingID, &s.SeatID, &s.PriceCents); err != nil {
			return nil, fmt.Errorf("scan booking seat: %w", err)
		}
		byBooking[bookingID] = append(byBooking[bookingID], s)
	}
	if err := seatRows.Err(); err != nil {
		return nil, err
	}
	for i := range bookings {
		bookings[i].Seats = byBooking[bookings[i].ID]
	}
	return bookings, nil
}

func (r *Repo) seats(ctx context.Context, bookingID string) ([]domain.Seat, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT seat_id, price_cents FROM booking_seats WHERE booking_id = $1`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("list seats: %w", err)
	}
	defer rows.Close()

	seats := []domain.Seat{}
	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.SeatID, &s.PriceCents); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		seats = append(seats, s)
	}
	return seats, rows.Err()
}

// Confirm marks a held booking as confirmed. The status check is part of the
// UPDATE, so a booking cannot be confirmed twice or after it expired; a caller
// that changes nothing gets ErrHoldExpired rather than a false success.
func (r *Repo) Confirm(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE bookings SET status = 'confirmed', updated_at = now()
		 WHERE id = $1 AND status = 'held' AND hold_expires_at > now()`, id)
	if err != nil {
		return fmt.Errorf("confirm booking: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrHoldExpired
	}
	return nil
}

// Release cancels or expires a booking and returns its seats to the pool.
func (r *Repo) Release(ctx context.Context, id, status string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin release: %w", err)
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE bookings SET status = $2, updated_at = now()
		 WHERE id = $1 AND status IN ('held', 'confirmed')`, id, status)
	if err != nil {
		return fmt.Errorf("release booking: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrBookingNotFound
	}
	if _, err := tx.Exec(ctx,
		`UPDATE booking_seats SET released_at = now()
		 WHERE booking_id = $1 AND released_at IS NULL`, id); err != nil {
		return fmt.Errorf("release seats: %w", err)
	}
	return tx.Commit(ctx)
}

// TakenSeats returns the seats of an event that are currently claimed, so the
// seat map can grey them out. Holds that have already expired are not counted.
func (r *Repo) TakenSeats(ctx context.Context, eventID string) ([]string, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT bs.seat_id
		 FROM booking_seats bs
		 JOIN bookings b ON b.id = bs.booking_id
		 WHERE bs.event_id = $1
		   AND bs.released_at IS NULL
		   AND (b.status = 'confirmed' OR (b.status = 'held' AND b.hold_expires_at > now()))`,
		eventID)
	if err != nil {
		return nil, fmt.Errorf("list taken seats: %w", err)
	}
	defer rows.Close()

	seats := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan taken seat: %w", err)
		}
		seats = append(seats, id)
	}
	return seats, rows.Err()
}

// ExpiredHold is a hold the sweeper has just released.
type ExpiredHold struct {
	BookingID string
	UserID    string
	UserEmail string
}

// SweepExpired expires every hold whose window has passed and returns them, so
// the caller can notify the users whose seats were released.
func (r *Repo) SweepExpired(ctx context.Context, now time.Time) ([]ExpiredHold, error) {
	rows, err := r.pool.Query(ctx, `
		WITH stale AS (
		    UPDATE bookings SET status = 'expired', updated_at = now()
		    WHERE status = 'held' AND hold_expires_at < $1
		    RETURNING id, user_id, user_email
		), released AS (
		    UPDATE booking_seats SET released_at = now()
		    WHERE booking_id IN (SELECT id FROM stale) AND released_at IS NULL
		)
		SELECT id, user_id, user_email FROM stale`, now)
	if err != nil {
		return nil, fmt.Errorf("sweep expired holds: %w", err)
	}
	defer rows.Close()

	var expired []ExpiredHold
	for rows.Next() {
		var e ExpiredHold
		if err := rows.Scan(&e.BookingID, &e.UserID, &e.UserEmail); err != nil {
			return nil, fmt.Errorf("scan expired hold: %w", err)
		}
		expired = append(expired, e)
	}
	return expired, rows.Err()
}
