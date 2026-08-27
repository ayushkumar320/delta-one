// Package repository stores auth entities in Postgres.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ayush/delta-one/services/auth/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo reads and writes users.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo returns a UserRepo backed by the given pool.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

const uniqueViolation = "23505"

// Create inserts a user. A duplicate email returns domain.ErrEmailTaken: the
// unique index decides, so two concurrent registrations cannot both succeed.
func (r *UserRepo) Create(ctx context.Context, u domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, role)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID, u.Email, u.PasswordHash, u.Name, u.Role)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return domain.ErrEmailTaken
	}
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

const selectUser = `SELECT id, email, password_hash, name, role, created_at FROM users`

// ByEmail looks a user up by address. The column is citext, so the comparison
// is case-insensitive.
func (r *UserRepo) ByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.one(ctx, selectUser+` WHERE email = $1`, email)
}

// ByID looks a user up by ID.
func (r *UserRepo) ByID(ctx context.Context, id string) (domain.User, error) {
	return r.one(ctx, selectUser+` WHERE id = $1`, id)
}

func (r *UserRepo) one(ctx context.Context, query string, args ...any) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, query, args...).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUserNotFound
	}
	if err != nil {
		return domain.User{}, fmt.Errorf("select user: %w", err)
	}
	return u, nil
}
