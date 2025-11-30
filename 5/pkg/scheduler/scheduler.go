package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/service"
)

type Scheduler struct {
	bookingService service.BookingService
	interval       time.Duration
}

func NewScheduler(bookingService service.BookingService, interval time.Duration) *Scheduler {
	return &Scheduler{
		bookingService: bookingService,
		interval:       interval,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.bookingService.CancelExpiredBookings(ctx); err != nil {
				fmt.Printf("Error canceling expired bookings: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
