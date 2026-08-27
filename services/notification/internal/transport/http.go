// Package transport exposes the notification service's health endpoint. The
// service does its real work as an event consumer, not over HTTP.
package transport

import (
	"net/http"

	"github.com/ayush/delta-one/shared/httpx"
	"github.com/ayush/delta-one/shared/middleware"
)

// Routes returns the service's router with shared middleware applied.
func Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "notification"})
	})
	return middleware.Chain(mux, middleware.Default("notification")...)
}
