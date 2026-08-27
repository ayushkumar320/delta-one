// Package middleware holds the HTTP middleware shared by every service:
// request IDs, access logging and panic recovery.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"time"
)

// Middleware wraps a handler with behaviour that runs around it.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware to h, outermost first, so Chain(h, A, B) runs A
// before B on the way in.
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

// Default is the middleware every service mounts: an ID on each request, a log
// line per response, and a recovered panic instead of a dropped connection.
func Default(service string) []Middleware {
	return []Middleware{RequestID, Recover, Logger(service)}
}

type ctxKey string

const requestIDKey ctxKey = "request_id"

// HeaderRequestID carries the request ID between services.
const HeaderRequestID = "X-Request-ID"

// RequestIDFrom returns the request ID stored in ctx, or "" when absent.
func RequestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// RequestID reuses an inbound request ID or generates one, so a single user
// action can be traced across services.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(HeaderRequestID)
		if id == "" {
			buf := make([]byte, 8)
			rand.Read(buf)
			id = hex.EncodeToString(buf)
		}
		w.Header().Set(HeaderRequestID, id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, id)))
	})
}

// statusRecorder captures the status code for the access log.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// Unwrap exposes the underlying writer to http.ResponseController, keeping
// flushing and hijacking available to handlers that need them.
func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }

// Logger writes one line per request: service, method, path, status, duration.
func Logger(service string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Printf("%s %s %s %d %s id=%s",
				service, r.Method, r.URL.Path, rec.status,
				time.Since(start).Round(time.Millisecond), RequestIDFrom(r.Context()))
		})
	}
}

// Recover turns a panicking handler into a 500 so one bad request cannot take
// the process down.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				log.Printf("panic: %v (%s %s id=%s)", v, r.Method, r.URL.Path, RequestIDFrom(r.Context()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":{"code":"internal","message":"something went wrong"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
