package service

import (
	"context"
	"log"
	"time"

	"github.com/ayush/delta-one/services/booking/internal/repository"
	"github.com/ayush/delta-one/shared/events"
)

// SweepInterval is how often abandoned holds are swept up. Holds are also
// released on demand inside the hold transaction, so this interval controls
// only how quickly the seat map stops showing an abandoned seat as taken and
// how quickly the customer is told, not whether seats are ever stuck.
const SweepInterval = 30 * time.Second

// Sweeper is the storage the sweeper needs.
type Sweeper interface {
	SweepExpired(ctx context.Context, now time.Time) ([]repository.ExpiredHold, error)
}

// RunSweeper expires abandoned holds until ctx is cancelled.
func RunSweeper(ctx context.Context, store Sweeper, publisher *events.Publisher) {
	ticker := time.NewTicker(SweepInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			expired, err := store.SweepExpired(ctx, time.Now())
			if err != nil {
				log.Printf("booking: sweep expired holds: %v", err)
				continue
			}
			for _, hold := range expired {
				log.Printf("booking: hold %s expired", hold.BookingID)
				if publisher == nil {
					continue
				}
				err := publisher.Publish(ctx, events.TypeBookingCancelled, events.BookingCancelled{
					BookingID: hold.BookingID,
					UserID:    hold.UserID,
					UserEmail: hold.UserEmail,
					Reason:    "the seat hold expired before payment",
				})
				if err != nil {
					log.Printf("booking: publish expiry for %s: %v", hold.BookingID, err)
				}
			}
		}
	}
}
