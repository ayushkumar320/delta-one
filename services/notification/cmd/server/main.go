// Command server runs the notification service: an event consumer that turns
// platform events into customer messages.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ayush/delta-one/services/notification/internal/service"
	"github.com/ayush/delta-one/services/notification/internal/transport"
	"github.com/ayush/delta-one/shared/config"
	"github.com/ayush/delta-one/shared/events"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	rdb, err := config.Redis(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer rdb.Close()

	notifier := service.New(service.LogSender{})

	// The consumer name distinguishes replicas within the group; the hostname
	// is the natural choice when several are running.
	consumer := events.NewConsumer(rdb, "notification", config.Env("HOSTNAME", "notification-1"))
	go func() {
		if err := consumer.Run(ctx, notifier.Handle); err != nil && ctx.Err() == nil {
			log.Fatalf("notification: consumer stopped: %v", err)
		}
	}()

	srv := &http.Server{
		Addr:              ":" + config.Env("NOTIFICATION_PORT", "8085"),
		Handler:           transport.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("notification listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()
	log.Print("notification shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("notification: shutdown: %v", err)
	}
}
