// Package transport routes public requests to the services behind the
// gateway, after authenticating and rate limiting them.
package transport

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"

	"github.com/ayush/delta-one/services/gateway/internal/service"
	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
	"github.com/ayush/delta-one/shared/token"
)

// Backends are the services the gateway routes to.
type Backends struct {
	Auth    string
	Catalog string
	Booking string
	Payment string
}

// Config is everything the gateway needs to serve.
type Config struct {
	Backends      Backends
	JWTSecret     string
	AllowedOrigin string
	Limiter       *service.Limiter
}

// Routes builds the gateway's router.
//
// Every public path is listed explicitly. A catch-all proxy would forward
// endpoints that were never meant to be public, such as the seat lookup the
// booking service calls on catalog.
func Routes(cfg Config) (http.Handler, error) {
	auth, err := proxy(cfg.Backends.Auth)
	if err != nil {
		return nil, err
	}
	catalog, err := proxy(cfg.Backends.Catalog)
	if err != nil {
		return nil, err
	}
	booking, err := proxy(cfg.Backends.Booking)
	if err != nil {
		return nil, err
	}
	payment, err := proxy(cfg.Backends.Payment)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "gateway"})
	})

	mux.Handle("POST /api/auth/register", auth)
	mux.Handle("POST /api/auth/login", auth)
	mux.Handle("GET /api/auth/me", auth)

	mux.Handle("GET /api/events", catalog)
	mux.Handle("POST /api/events", catalog)
	mux.Handle("GET /api/events/{id}", catalog)
	mux.Handle("GET /api/events/{id}/seats", catalog)
	mux.Handle("GET /api/venues", catalog)
	mux.Handle("POST /api/venues", catalog)

	// More specific than GET /api/events/{id}/..., so the mux prefers it.
	mux.Handle("GET /api/events/{id}/taken-seats", booking)
	mux.Handle("POST /api/bookings", booking)
	mux.Handle("GET /api/bookings", booking)
	mux.Handle("GET /api/bookings/{id}", booking)
	mux.Handle("POST /api/bookings/{id}/confirm", booking)
	mux.Handle("DELETE /api/bookings/{id}", booking)

	mux.Handle("GET /api/payments/{id}", payment)

	handler := middleware.Chain(mux,
		append(middleware.Default("gateway"),
			cors(cfg.AllowedOrigin),
			rateLimit(cfg.Limiter),
			authenticate(cfg.JWTSecret),
		)...)
	return handler, nil
}

// proxy forwards to one backend, stripping the /api prefix the public API uses
// but the services do not.
func proxy(target string) (http.Handler, error) {
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(parsed)
	director := rp.Director
	rp.Director = func(r *http.Request) {
		director(r)
		r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
		r.Host = parsed.Host
	}
	return rp, nil
}

// authenticate verifies the bearer token and republishes the caller's identity
// as headers for the service behind the gateway.
//
// Inbound identity headers are always deleted first. Without that, anyone could
// send X-User-ID and be trusted by every service in the platform.
func authenticate(secret string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Header.Del(middleware.HeaderUserID)
			r.Header.Del(middleware.HeaderUserEmail)
			r.Header.Del(middleware.HeaderRole)

			raw := bearer(r)
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}
			claims, err := token.Verify(secret, raw)
			if err != nil {
				httpx.Fail(w, httpx.Unauthorized("invalid_token", "your session has expired; sign in again"))
				return
			}
			r.Header.Set(middleware.HeaderUserID, claims.Subject)
			r.Header.Set(middleware.HeaderUserEmail, claims.Email)
			r.Header.Set(middleware.HeaderRole, claims.Role)
			next.ServeHTTP(w, r)
		})
	}
}

func bearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	value, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

// rateLimit caps requests per client IP.
func rateLimit(limiter *service.Limiter) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}
			allowed, remaining := limiter.Allow(r.Context(), clientIP(r))
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limiter.Limit()))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
			if !allowed {
				httpx.Fail(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited",
					"too many requests; slow down and try again shortly"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP is the peer address. X-Forwarded-For is deliberately ignored: this
// gateway is the edge, and trusting a client-supplied header would let anyone
// spoof their way around the limit.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// cors allows the browser frontend to call the API from its own origin.
func cors(origin string) middleware.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin != "" && r.Header.Get("Origin") == origin {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, "+middleware.HeaderRequestID)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
