package domain

import "testing"

func TestValidateRegistration(t *testing.T) {
	tests := []struct {
		name                        string
		email, password, user, role string
		wantErr                     bool
	}{
		{"valid", "ada@example.com", "correct-horse", "Ada", RoleCustomer, false},
		{"bad email", "ada-at-example", "correct-horse", "Ada", RoleCustomer, true},
		{"short password", "ada@example.com", "short", "Ada", RoleCustomer, true},
		{"blank name", "ada@example.com", "correct-horse", "   ", RoleCustomer, true},
		{"unknown role", "ada@example.com", "correct-horse", "Ada", "admin", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegistration(tt.email, tt.password, tt.user, tt.role)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Ada@Example.COM "); got != "ada@example.com" {
		t.Fatalf("got %q", got)
	}
}
