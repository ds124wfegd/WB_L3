package repository

import (
	"context"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

type BookingRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, booking *entity.Booking) error
	GetByID(ctx context.Context, id int64) (*entity.Booking, error)
	GetByEventAndUser(ctx context.Context, eventID, userID int64) (*entity.Booking, error)
	UpdateStatus(ctx context.Context, id int64, status entity.BookingStatus) error
	Update(ctx context.Context, booking *entity.Booking) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	GetByEventID(ctx context.Context, eventID int64) ([]*entity.Booking, error)
	GetByUserID(ctx context.Context, userID int64) ([]*entity.Booking, error)
	GetByStatus(ctx context.Context, status entity.BookingStatus) ([]*entity.Booking, error)
	GetByEventAndStatus(ctx context.Context, eventID int64, status entity.BookingStatus) ([]*entity.Booking, error)

	// Expiration operations
	GetExpiredBookings(ctx context.Context, before time.Time) ([]*entity.BookingExpiration, error)
	GetExpiringBookings(ctx context.Context, from, to time.Time) ([]*entity.BookingExpiration, error)
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	BulkUpdateStatus(ctx context.Context, ids []int64, status entity.BookingStatus) error

	// Statistical operations
	CountByEvent(ctx context.Context, eventID int64) (int, error)
	CountByEventAndStatus(ctx context.Context, eventID int64, status entity.BookingStatus) (int, error)
	GetEventBookingStats(ctx context.Context, eventID int64) (*entity.EventBookingStats, error)

	// Locking operations for concurrency control
	LockBooking(ctx context.Context, id int64) error
	GetWithLock(ctx context.Context, id int64) (*entity.Booking, error)

	GetAll(ctx context.Context) ([]*entity.Booking, error)
	GetRecentBookings(ctx context.Context, limit int) ([]*entity.Booking, error)
}

type EventRepository interface {
	Create(ctx context.Context, event *entity.Event) error
	GetByID(ctx context.Context, id int64) (*entity.EventWithAvailability, error)
	GetAll(ctx context.Context) ([]*entity.EventWithAvailability, error)

	// CRUD операции

	Update(ctx context.Context, event *entity.Event) error
	Delete(ctx context.Context, id int64) error

	// Статистика и дополнительные методы
	GetEventsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.Event, error)
	GetUpcomingEvents(ctx context.Context, limit int) ([]*entity.EventWithAvailability, error)
	SearchByTitle(ctx context.Context, title string) ([]*entity.EventWithAvailability, error)
	UpdateSeats(ctx context.Context, eventID int64, seats int) error
}

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id int64) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error)
	UpdateTelegramID(ctx context.Context, userID int64, telegramID string) error

	// CRUD операции
	Update(ctx context.Context, user *entity.User) error
	Delete(ctx context.Context, id int64) error

	// Дополнительные методы
	GetAll(ctx context.Context) ([]*entity.User, error)
	SearchByName(ctx context.Context, name string) ([]*entity.User, error)
}
