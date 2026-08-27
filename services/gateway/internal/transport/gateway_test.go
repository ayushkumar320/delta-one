package transport

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ayush/delta-one/shared/middleware"
	"github.com/ayush/delta-one/shared/token"
)

// echoBackend reports the identity headers and path it received.
func echoBackend(t *testing.T) (*httptest.Server, *http.Header, *string) {
	t.Helper()
	var seen http.Header
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Clone()
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &seen, &path
}

func newGateway(t *testing.T, backend string) http.Handler {
	t.Helper()
	handler, err := Routes(Config{
		Backends:  Backends{Auth: backend, Catalog: backend, Booking: backend, Payment: backend},
		JWTSecret: "s3cret",
	})
	if err != nil {
		t.Fatal(err)
	}
	return handler
}

func TestGatewayStripsSpoofedIdentityHeaders(t *testing.T) {
	backend, seen, _ := echoBackend(t)
	gateway := newGateway(t, backend.URL)

	req := httptest.NewRequest("GET", "/api/events", nil)
	req.Header.Set(middleware.HeaderUserID, "victim-user-id")
	req.Header.Set(middleware.HeaderRole, "organizer")
	gateway.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen.Get(middleware.HeaderUserID); got != "" {
		t.Fatalf("client-supplied user id reached the backend: %q", got)
	}
	if got := seen.Get(middleware.HeaderRole); got != "" {
		t.Fatalf("client-supplied role reached the backend: %q", got)
	}
}

func TestGatewayForwardsVerifiedIdentity(t *testing.T) {
	backend, seen, path := echoBackend(t)
	gateway := newGateway(t, backend.URL)

	signed, _, err := token.Issue("s3cret", "user-1", "ada@example.com", "organizer")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/api/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+signed)
	gateway.ServeHTTP(httptest.NewRecorder(), req)

	if got := seen.Get(middleware.HeaderUserID); got != "user-1" {
		t.Fatalf("user id = %q, want user-1", got)
	}
	if got := seen.Get(middleware.HeaderRole); got != "organizer" {
		t.Fatalf("role = %q, want organizer", got)
	}
	if *path != "/bookings" {
		t.Fatalf("backend path = %q, want /bookings without the /api prefix", *path)
	}
}

func TestGatewayRejectsATamperedToken(t *testing.T) {
	backend, _, _ := echoBackend(t)
	gateway := newGateway(t, backend.URL)

	signed, _, _ := token.Issue("a-different-secret", "user-1", "ada@example.com", "customer")
	req := httptest.NewRequest("GET", "/api/bookings", nil)
	req.Header.Set("Authorization", "Bearer "+signed)

	w := httptest.NewRecorder()
	gateway.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestGatewayDoesNotExposeInternalEndpoints(t *testing.T) {
	backend, _, _ := echoBackend(t)
	gateway := newGateway(t, backend.URL)

	// The booking service calls this on catalog; the public must not.
	req := httptest.NewRequest("POST", "/api/events/event-1/seats/lookup", nil)
	w := httptest.NewRecorder()
	gateway.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 for an unrouted internal endpoint", w.Code)
	}
}
