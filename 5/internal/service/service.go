package service

import (
	"context"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

type EventService interface {
	// Основные операции
	CreateEvent(ctx context.Context, req *CreateEventRequest) (*entity.Event, error)
	GetEvent(ctx context.Context, id int64) (*entity.EventWithAvailability, error)
	GetAllEvents(ctx context.Context) ([]*entity.EventWithAvailability, error)
	UpdateEvent(ctx context.Context, id int64, req *UpdateEventRequest) (*entity.Event, error)
	DeleteEvent(ctx context.Context, id int64) error

	// Дополнительные операции
	GetEventBookings(ctx context.Context, eventID int64) ([]*entity.Booking, error)
	GetEventStats(ctx context.Context, eventID int64) (*entity.EventStats, error)
	SearchEvents(ctx context.Context, filter *EventFilter) ([]*entity.EventWithAvailability, error)
	GetUpcomingEvents(ctx context.Context, limit int) ([]*entity.EventWithAvailability, error)
	SearchEventsByTitle(ctx context.Context, title string) ([]*entity.EventWithAvailability, error)
}

// UserService defines the interface for user operations
type UserService interface {
	// Основные операции
	RegisterUser(ctx context.Context, req *RegisterUserRequest) (*entity.User, error)
	GetUserByID(ctx context.Context, id int64) (*entity.User, error) // ДОБАВЛЕНО
	GetUserByEmail(ctx context.Context, email string) (*entity.User, error)
	UpdateUser(ctx context.Context, id int64, req *UpdateUserRequest) (*entity.User, error)
	LinkTelegram(ctx context.Context, userID int64, telegramID string) error
	DeleteUser(ctx context.Context, id int64) error

	// Статистика и аналитика
	GetUserStats(ctx context.Context, userID int64) (*UserStats, error)

	// Поиск и списки
	GetAllUsers(ctx context.Context) ([]*entity.User, error)
	SearchUsersByName(ctx context.Context, name string) ([]*entity.User, error)
}

// BookingService определяет интерфейс для операций с бронированиями
type BookingService interface {
	// Основные операции
	BookSeats(ctx context.Context, req *BookSeatsRequest) (*entity.Booking, error)
	ConfirmBooking(ctx context.Context, bookingID int64) error
	CancelBooking(ctx context.Context, bookingID int64, reason string) error
	GetBooking(ctx context.Context, id int64) (*entity.Booking, error)
	GetUserBookings(ctx context.Context, userID int64) ([]*entity.Booking, error)
	GetEventBookings(ctx context.Context, eventID int64) ([]*entity.Booking, error)

	// Операции истечения срока
	CancelExpiredBookings(ctx context.Context) error
	GetExpiredBookings(ctx context.Context, before time.Time) ([]*entity.BookingExpiration, error)
	ExpireBooking(ctx context.Context, bookingID int64) error

	// Дополнительные операции
	GetBookingsByStatus(ctx context.Context, status entity.BookingStatus) ([]*entity.Booking, error)
	UpdateBookingSeats(ctx context.Context, bookingID int64, seats int) error
	UpdateBookingStatus(ctx context.Context, bookingID int64, status entity.BookingStatus) error
	GetBookingStats(ctx context.Context) (*BookingStats, error)

	// Административные операции
	GetAllBookings(ctx context.Context) ([]*entity.Booking, error)
	DeleteBooking(ctx context.Context, bookingID int64) error
	GetRecentBookings(ctx context.Context, limit int) ([]*entity.Booking, error)

	// Утилиты
	GetBookingWithDetails(ctx context.Context, bookingID int64) (*BookingDetails, error)
	CheckBookingAvailability(ctx context.Context, eventID int64, seats int) (bool, error)
}
