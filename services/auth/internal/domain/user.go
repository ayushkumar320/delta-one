// Package domain holds the auth entities and the rules that govern them. It
// imports no other layer of the service.
package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/ayush/delta-one/shared/httpx"
)

// Roles a user can hold. Organizers may create events; customers may not.
const (
	RoleCustomer  = "customer"
	RoleOrganizer = "organizer"
)

// MinPasswordLength is the shortest password accepted at registration. Length
// is the only rule: composition rules push users towards predictable passwords.
const MinPasswordLength = 10

// User is a registered account. PasswordHash never leaves the service.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Name         string
	Role         string
	CreatedAt    time.Time
}

// Errors the auth domain returns. The credentials error is deliberately vague:
// saying which half was wrong tells an attacker which emails are registered.
var (
	ErrEmailTaken   = httpx.Conflict("email_taken", "that email is already registered")
	ErrCredentials  = httpx.Unauthorized("invalid_credentials", "email or password is incorrect")
	ErrUserNotFound = httpx.NotFound("user_not_found", "user not found")
)

// NormalizeEmail trims and lowercases an address so lookups and the unique
// index agree on what counts as the same account.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateRegistration checks a registration request before any hashing or
// database work happens.
func ValidateRegistration(email, password, name, role string) error {
	if _, err := mail.ParseAddress(email); err != nil {
		return httpx.BadRequest("invalid_email", "enter a valid email address")
	}
	if len(password) < MinPasswordLength {
		return httpx.BadRequest("weak_password",
			"password must be at least 10 characters")
	}
	if strings.TrimSpace(name) == "" {
		return httpx.BadRequest("missing_name", "name is required")
	}
	if role != RoleCustomer && role != RoleOrganizer {
		return httpx.BadRequest("invalid_role", "role must be customer or organizer")
	}
	return nil
}
