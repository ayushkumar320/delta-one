// Command server runs the API gateway: the single public entry point that
// authenticates, rate limits and routes to the services behind it.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayush/delta-one/services/gateway/internal/service"
	"github.com/ayush/delta-one/services/gateway/internal/transport"
	"github.com/ayush/delta-one/shared/config"
)

// requestsPerMinute is the per-IP ceiling. Generous enough for a person
// browsing a seat map, tight enough to blunt a scripted flood.
const requestsPerMinute = 120

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rdb, err := config.Redis(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	handler, err := transport.Routes(transport.Config{
		Backends: transport.Backends{
			Auth:    config.Env("AUTH_URL", "http://localhost:8081"),
			Catalog: config.Env("CATALOG_URL", "http://localhost:8082"),
			Booking: config.Env("BOOKING_URL", "http://localhost:8083"),
			Payment: config.Env("PAYMENT_URL", "http://localhost:8084"),
		},
		JWTSecret:     config.MustEnv("JWT_SECRET"),
		AllowedOrigin: config.Env("FRONTEND_ORIGIN", "http://localhost:5173"),
		Limiter:       service.NewLimiter(rdb, requestsPerMinute, time.Minute),
	})
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:              ":" + config.Env("GATEWAY_PORT", "8080"),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("gateway listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("gateway shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("gateway: shutdown: %v", err)
	}
}
