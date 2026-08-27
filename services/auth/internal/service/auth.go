// Package service holds the auth business logic: registration, login and
// profile lookup.
package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/ayush/delta-one/services/auth/internal/domain"
	"github.com/ayush/delta-one/shared/events"
	"github.com/ayush/delta-one/shared/token"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Users is the storage the service needs. Declared here, in the consumer, so
// the service can be tested without Postgres.
type Users interface {
	Create(ctx context.Context, u domain.User) error
	ByEmail(ctx context.Context, email string) (domain.User, error)
	ByID(ctx context.Context, id string) (domain.User, error)
}

// Auth issues credentials for accounts.
type Auth struct {
	users     Users
	publisher *events.Publisher
	secret    string
}

// New returns an Auth service. publisher may be nil, in which case registration
// events are not published.
func New(users Users, publisher *events.Publisher, secret string) *Auth {
	return &Auth{users: users, publisher: publisher, secret: secret}
}

// Session is a successful authentication: the account and its access token.
type Session struct {
	User      domain.User
	Token     string
	ExpiresAt time.Time
}

// Register creates an account and signs the user in.
func (a *Auth) Register(ctx context.Context, email, password, name, role string) (Session, error) {
	if role == "" {
		role = domain.RoleCustomer
	}
	email = domain.NormalizeEmail(email)
	if err := domain.ValidateRegistration(email, password, name, role); err != nil {
		return Session{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return Session{}, fmt.Errorf("hash password: %w", err)
	}

	user := domain.User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: string(hash),
		Name:         name,
		Role:         role,
	}
	if err := a.users.Create(ctx, user); err != nil {
		return Session{}, err
	}
	user.CreatedAt = time.Now().UTC()

	a.publish(ctx, events.TypeUserRegistered, events.UserRegistered{
		UserID: user.ID, Email: user.Email, Name: user.Name,
	})
	return a.session(user)
}

// dummyHash is compared against when no account matches, so a login attempt
// costs the same whether or not the email exists. Without it, response time
// reveals which addresses are registered.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer"), bcrypt.DefaultCost)

// Login verifies credentials and issues an access token.
func (a *Auth) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := a.users.ByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil {
		if errors.Is(err, domain.ErrUserNotFound) {
			bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return Session{}, domain.ErrCredentials
		}
		return Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return Session{}, domain.ErrCredentials
	}
	return a.session(user)
}

// Profile returns the account behind a user ID.
func (a *Auth) Profile(ctx context.Context, userID string) (domain.User, error) {
	return a.users.ByID(ctx, userID)
}

func (a *Auth) session(user domain.User) (Session, error) {
	signed, expiry, err := token.Issue(a.secret, user.ID, user.Email, user.Role)
	if err != nil {
		return Session{}, fmt.Errorf("issue token: %w", err)
	}
	return Session{User: user, Token: signed, ExpiresAt: expiry}, nil
}

// publish reports failures without failing the request: the account exists
// whether or not the welcome email goes out.
func (a *Auth) publish(ctx context.Context, eventType string, payload any) {
	if a.publisher == nil {
		return
	}
	if err := a.publisher.Publish(ctx, eventType, payload); err != nil {
		log.Printf("auth: publish %s: %v", eventType, err)
	}
}
