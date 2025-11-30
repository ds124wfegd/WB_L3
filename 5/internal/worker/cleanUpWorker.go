package worker

import (
	"context"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/service"

	"github.com/sirupsen/logrus"
)

type BookingCleanupWorker struct {
	bookingService service.BookingService
	interval       time.Duration
}

func NewBookingCleanupWorker(bookingService service.BookingService, interval time.Duration) *BookingCleanupWorker {
	return &BookingCleanupWorker{
		bookingService: bookingService,
		interval:       interval,
	}
}

func (w *BookingCleanupWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	logrus.Info("Booking cleanup worker started")

	for {
		select {
		case <-ctx.Done():
			logrus.Info("Booking cleanup worker stopped")
			return
		case <-ticker.C:
			w.cleanupExpiredBookings(ctx)
		}
	}
}

// cleanupExpiredBookings выполняет очистку истекших бронирований
func (w *BookingCleanupWorker) cleanupExpiredBookings(ctx context.Context) {
	logrus.Info("Starting expired bookings cleanup")

	// Получаем текущее время для фильтрации
	now := time.Now()

	// Получаем список истекших бронирований
	expiredBookings, err := w.bookingService.GetExpiredBookings(ctx, now)
	if err != nil {
		logrus.Errorf("Failed to get expired bookings: %v", err)
		return
	}

	if len(expiredBookings) == 0 {
		logrus.Info("No expired bookings found for cleanup")
		return
	}

	logrus.Infof("Found %d expired bookings for cleanup", len(expiredBookings))

	// Обрабатываем каждое истекшее бронирование
	successCount := 0
	failedCount := 0

	for _, expired := range expiredBookings {
		// Проверяем, не был ли контекст отменен во время обработки
		select {
		case <-ctx.Done():
			logrus.Info("Cleanup interrupted by context cancellation")
			return
		default:
			// Продолжаем обработку
		}

		// Помечаем бронирование как истекшее
		err := w.bookingService.ExpireBooking(ctx, expired.BookingID)
		if err != nil {
			logrus.Errorf("Failed to expire booking %d: %v", expired.BookingID, err)
			failedCount++
			continue
		}

		logrus.Debugf("Successfully expired booking %d for event '%s'",
			expired.BookingID, expired.EventTitle)
		successCount++
	}

	// Логируем результаты очистки
	logrus.Infof("Expired bookings cleanup completed: %d successful, %d failed",
		successCount, failedCount)

	// Если есть неудачные попытки, логируем предупреждение
	if failedCount > 0 {
		logrus.Warnf("%d bookings failed to expire during cleanup", failedCount)
	}

	// Дополнительно: выполняем массовую отмену истекших бронирований через сервис
	w.performBulkCancellation(ctx)
}

// performBulkCancellation выполняет массовую отмену истекших бронирований
func (w *BookingCleanupWorker) performBulkCancellation(ctx context.Context) {
	logrus.Info("Starting bulk cancellation of expired bookings")

	err := w.bookingService.CancelExpiredBookings(ctx)
	if err != nil {
		logrus.Errorf("Failed to perform bulk cancellation of expired bookings: %v", err)
		return
	}

	logrus.Info("Bulk cancellation of expired bookings completed successfully")
}

// Stop останавливает воркер (дополнительный метод для graceful shutdown)
func (w *BookingCleanupWorker) Stop() {
	logrus.Info("Booking cleanup worker stopping...")
}

// GetStats возвращает статистику работы воркера (дополнительный метод)
func (w *BookingCleanupWorker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"worker_type": "booking_cleanup",
		"interval":    w.interval.String(),
		"status":      "running",
	}
}
