// Command server runs the catalog service: venues, events and seat maps.
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
	"github.com/ayush/delta-one/services/catalog/internal/repository"
	"github.com/ayush/delta-one/services/catalog/internal/service"
	"github.com/ayush/delta-one/services/catalog/internal/transport"
	"github.com/ayush/delta-one/shared/config"
	"github.com/ayush/delta-one/shared/migrate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := config.Postgres(ctx, config.Env("CATALOG_DB_NAME", "delta_catalog"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, migrations.Files, "catalog"); err != nil {
		log.Fatal(err)
	}

	catalog := service.New(repository.New(pool))

	srv := &http.Server{
		Addr:              ":" + config.Env("CATALOG_PORT", "8082"),
		Handler:           transport.NewHandler(catalog).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("catalog listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("catalog shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("catalog: shutdown: %v", err)
	}
}
