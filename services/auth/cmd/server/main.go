// Command server runs the auth service: registration, login and token issuing.
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
	"github.com/ayush/delta-one/services/auth/internal/repository"
	"github.com/ayush/delta-one/services/auth/internal/service"
	"github.com/ayush/delta-one/services/auth/internal/transport"
	"github.com/ayush/delta-one/shared/config"
	"github.com/ayush/delta-one/shared/events"
	"github.com/ayush/delta-one/shared/migrate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	secret := config.MustEnv("JWT_SECRET")

	pool, err := config.Postgres(ctx, config.Env("AUTH_DB_NAME", "delta_auth"))
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, migrations.Files, "auth"); err != nil {
		log.Fatal(err)
	}

	rdb, err := config.Redis(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	auth := service.New(repository.NewUserRepo(pool), events.NewPublisher(rdb), secret)

	srv := &http.Server{
		Addr:              ":" + config.Env("AUTH_PORT", "8081"),
		Handler:           transport.NewHandler(auth).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("auth listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("auth shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("auth: shutdown: %v", err)
	}
}
