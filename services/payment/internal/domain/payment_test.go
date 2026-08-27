package domain

import "testing"

func TestAuthorize(t *testing.T) {
	if reason := Authorize("tok_visa", 250000); reason != "" {
		t.Fatalf("ordinary charge failed: %s", reason)
	}
	if Authorize(TokenDecline, 250000) == "" {
		t.Fatal("decline token was authorized")
	}
	if Authorize(TokenInsufficient, 250000) == "" {
		t.Fatal("insufficient-funds token was authorized")
	}
	if Authorize("tok_visa", 9_000_000) == "" {
		t.Fatal("charge above the limit was authorized")
	}
}

func TestValidateCharge(t *testing.T) {
	if err := ValidateCharge("b1", "u1", "tok_visa", 100); err != nil {
		t.Fatalf("valid charge rejected: %v", err)
	}
	if err := ValidateCharge("b1", "u1", "tok_visa", 0); err == nil {
		t.Fatal("zero amount accepted")
	}
	if err := ValidateCharge("b1", "u1", "4111111111111111", 100); err == nil {
		t.Fatal("raw card number accepted")
	}
}
