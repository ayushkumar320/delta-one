// Package repository stores payments in Postgres.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ayush/delta-one/services/payment/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repo reads and writes payments.
type Repo struct {
	pool *pgxpool.Pool
}

// New returns a Repo backed by the given pool.
func New(pool *pgxpool.Pool) *Repo { return &Repo{pool: pool} }

const uniqueViolation = "23505"

// ErrDuplicate reports that a succeeded payment for this booking already
// exists. The unique index decides, so two concurrent charges cannot both win.
var ErrDuplicate = errors.New("payment already recorded")

// Create inserts a payment attempt.
func (r *Repo) Create(ctx context.Context, p domain.Payment) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO payments (id, booking_id, user_id, amount_cents, status, failure_reason)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.BookingID, p.UserID, p.AmountCents, p.Status, p.FailureReason)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}
	return nil
}

const selectPayment = `SELECT id, booking_id, user_id, amount_cents, status, failure_reason, created_at
	FROM payments`

// SucceededByBooking returns the successful payment for a booking, if any.
func (r *Repo) SucceededByBooking(ctx context.Context, bookingID string) (domain.Payment, error) {
	return r.one(ctx, selectPayment+` WHERE booking_id = $1 AND status = 'succeeded'`, bookingID)
}

// ByID returns one payment.
func (r *Repo) ByID(ctx context.Context, id string) (domain.Payment, error) {
	return r.one(ctx, selectPayment+` WHERE id = $1`, id)
}

func (r *Repo) one(ctx context.Context, query string, args ...any) (domain.Payment, error) {
	var p domain.Payment
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&p.ID, &p.BookingID, &p.UserID, &p.AmountCents, &p.Status, &p.FailureReason, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Payment{}, domain.ErrPaymentNotFound
	}
	if err != nil {
		return domain.Payment{}, fmt.Errorf("select payment: %w", err)
	}
	return p, nil
}
