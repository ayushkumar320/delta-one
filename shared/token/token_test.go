package token

import (
	"strings"
	"testing"
)

func TestIssueThenVerify(t *testing.T) {
	signed, _, err := Issue("s3cret", "user-1", "a@example.com", "customer")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := Verify("s3cret", signed)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "user-1" || claims.Email != "a@example.com" || claims.Role != "customer" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyRejectsWrongSecret(t *testing.T) {
	signed, _, _ := Issue("s3cret", "user-1", "a@example.com", "customer")
	if _, err := Verify("other", signed); err == nil {
		t.Fatal("token verified under the wrong secret")
	}
}

func TestVerifyRejectsAlgNone(t *testing.T) {
	// header {"alg":"none","typ":"JWT"} with the signature stripped.
	const header = "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0"
	signed, _, _ := Issue("s3cret", "user-1", "a@example.com", "customer")
	parts := strings.Split(signed, ".")
	if _, err := Verify("s3cret", header+"."+parts[1]+"."); err == nil {
		t.Fatal("alg=none token was accepted")
	}
}
