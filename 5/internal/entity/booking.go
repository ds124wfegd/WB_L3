package entity

import (
	"time"
)

type BookingStatus string

const (
	BookingStatusPending   BookingStatus = "pending"
	BookingStatusConfirmed BookingStatus = "confirmed"
	BookingStatusCancelled BookingStatus = "cancelled"
	BookingStatusExpired   BookingStatus = "expired"
)

type Booking struct {
	ID                 int64         `json:"id" db:"id"`
	EventID            int64         `json:"event_id" db:"event_id"`
	UserID             int64         `json:"user_id" db:"user_id"`
	Seats              int           `json:"seats" db:"seats"`
	Status             BookingStatus `json:"status" db:"status"`
	ExpiresAt          time.Time     `json:"expires_at" db:"expires_at"`
	ReservationTimeout int           `json:"reservation_timeout" db:"reservation_timeout"`
	CreatedAt          time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
}

type BookingExpiration struct {
	BookingID  int64     `json:"booking_id"`
	ExpiresAt  time.Time `json:"expires_at"`
	UserID     int64     `json:"user_id"`
	EventID    int64     `json:"event_id"`
	TelegramID string    `json:"telegram_id"`
	UserName   string    `json:"user_name"`
	EventTitle string    `json:"event_title"`
	Seats      int       `json:"seats"`
}
