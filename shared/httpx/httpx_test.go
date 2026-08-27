package httpx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFailMapsDomainErrors(t *testing.T) {
	w := httptest.NewRecorder()
	Fail(w, fmt.Errorf("wrapped: %w", NotFound("no_event", "event not found")))

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
	var body errorBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "no_event" {
		t.Fatalf("code = %q, want no_event", body.Error.Code)
	}
}

func TestFailHidesUnknownErrors(t *testing.T) {
	w := httptest.NewRecorder()
	Fail(w, errors.New("connection string: postgres://user:hunter2@db"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	if strings.Contains(w.Body.String(), "hunter2") {
		t.Fatal("internal error detail leaked to the client")
	}
}

func TestDecodeRejectsUnknownFields(t *testing.T) {
	r := httptest.NewRequest("POST", "/", strings.NewReader(`{"emial":"typo@example.com"}`))
	var payload struct {
		Email string `json:"email"`
	}
	if err := Decode(r, &payload); err == nil {
		t.Fatal("want error for unknown field, got nil")
	}
}
