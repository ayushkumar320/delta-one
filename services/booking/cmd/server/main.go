// Command server runs the booking service: seat holds, confirmation and
// cancellation.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayush/delta-one/migrations"
	"github.com/ayush/delta-one/services/booking/internal/client"
	"github.com/ayush/delta-one/services/booking/internal/repository"
	"github.com/ayush/delta-one/services/booking/internal/service"
	"github.com/ayush/delta-one/services/booking/internal/transport"
	"github.com/ayush/delta-one/shared/config"
	"github.com/ayush/delta-one/shared/events"
	"github.com/ayush/delta-one/shared/migrate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := config.Postgres(ctx, config.Env("BOOKING_DB_NAME", "delta_booking"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, migrations.Files, "booking"); err != nil {
		log.Fatal(err)
	}

	rdb, err := config.Redis(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	store := repository.New(pool)
	publisher := events.NewPublisher(rdb)
	booking := service.New(store,
		client.NewCatalog(config.Env("CATALOG_URL", "http://localhost:8082")),
		client.NewPayment(config.Env("PAYMENT_URL", "http://localhost:8084")),
		publisher)

	go service.RunSweeper(ctx, store, publisher)

	srv := &http.Server{
		Addr:              ":" + config.Env("BOOKING_PORT", "8083"),
		Handler:           transport.NewHandler(booking).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("booking listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("booking shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("booking: shutdown: %v", err)
	}
}
