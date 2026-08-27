package service

import (
	"context"
	"testing"

	"github.com/ayush/delta-one/services/auth/internal/domain"
	"github.com/ayush/delta-one/shared/token"
)

// memUsers is an in-memory Users for tests.
type memUsers map[string]domain.User

func (m memUsers) Create(_ context.Context, u domain.User) error {
	if _, ok := m[u.Email]; ok {
		return domain.ErrEmailTaken
	}
	m[u.Email] = u
	return nil
}

func (m memUsers) ByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := m[email]
	if !ok {
		return domain.User{}, domain.ErrUserNotFound
	}
	return u, nil
}

func (m memUsers) ByID(_ context.Context, id string) (domain.User, error) {
	for _, u := range m {
		if u.ID == id {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrUserNotFound
}

func TestRegisterThenLogin(t *testing.T) {
	auth := New(memUsers{}, nil, "s3cret")
	ctx := context.Background()

	registered, err := auth.Register(ctx, "Ada@Example.com", "correct-horse", "Ada", "")
	if err != nil {
		t.Fatal(err)
	}
	if registered.User.Role != domain.RoleCustomer {
		t.Fatalf("role = %q, want customer", registered.User.Role)
	}
	if registered.User.PasswordHash == "correct-horse" {
		t.Fatal("password stored in plain text")
	}

	claims, err := token.Verify("s3cret", registered.Token)
	if err != nil {
		t.Fatalf("issued token does not verify: %v", err)
	}
	if claims.Subject != registered.User.ID {
		t.Fatalf("subject = %q, want %q", claims.Subject, registered.User.ID)
	}

	// The address was normalized at registration, so a differently cased
	// login must still find the account.
	if _, err := auth.Login(ctx, "ADA@example.com", "correct-horse"); err != nil {
		t.Fatalf("login with normalized email: %v", err)
	}
	if _, err := auth.Login(ctx, "ada@example.com", "wrong-password"); err == nil {
		t.Fatal("login succeeded with the wrong password")
	}
	if _, err := auth.Login(ctx, "nobody@example.com", "correct-horse"); err != domain.ErrCredentials {
		t.Fatalf("unknown account error = %v, want ErrCredentials", err)
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	auth := New(memUsers{}, nil, "s3cret")
	ctx := context.Background()
	if _, err := auth.Register(ctx, "ada@example.com", "correct-horse", "Ada", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := auth.Register(ctx, "ada@example.com", "correct-horse", "Ada", ""); err != domain.ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}
