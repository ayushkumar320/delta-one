// Command server runs the payment service: a simulated card gateway.
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
	"github.com/ayush/delta-one/services/payment/internal/repository"
	"github.com/ayush/delta-one/services/payment/internal/service"
	"github.com/ayush/delta-one/services/payment/internal/transport"
	"github.com/ayush/delta-one/shared/config"
	"github.com/ayush/delta-one/shared/events"
	"github.com/ayush/delta-one/shared/migrate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := config.Postgres(ctx, config.Env("PAYMENT_DB_NAME", "delta_payment"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, migrations.Files, "payment"); err != nil {
		log.Fatal(err)
	}

	rdb, err := config.Redis(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	payments := service.New(repository.New(pool), events.NewPublisher(rdb))

	srv := &http.Server{
		Addr:              ":" + config.Env("PAYMENT_PORT", "8084"),
		Handler:           transport.NewHandler(payments).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("payment listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("payment shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("payment: shutdown: %v", err)
	}
}
