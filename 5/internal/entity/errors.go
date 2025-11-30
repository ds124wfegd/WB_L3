package entity

import "errors"

var (
	// Event errors
	ErrEventNotFound      = errors.New("event not found")
	ErrEventAlreadyExists = errors.New("event already exists")
	ErrEventFull          = errors.New("event is full")
	ErrEventDatePast      = errors.New("event date cannot be in the past")

	// Booking errors
	ErrBookingNotFound      = errors.New("booking not found")
	ErrBookingAlreadyExists = errors.New("booking already exists")
	ErrNotEnoughSeats       = errors.New("not enough available seats")
	ErrBookingExpired       = errors.New("booking has expired")
	ErrInvalidBookingStatus = errors.New("invalid booking status")

	// User errors
	ErrUserNotFound      = errors.New("user not found")
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrInvalidEmail      = errors.New("invalid email format")
	ErrTelegramIDExists  = errors.New("telegram ID already exists")

	// General errors
	ErrInvalidInput     = errors.New("invalid input")
	ErrDatabaseError    = errors.New("database error")
	ErrConcurrentUpdate = errors.New("concurrent update detected")
	ErrUnauthorized     = errors.New("unauthorized access")
	ErrForbidden        = errors.New("forbidden operation")
)
