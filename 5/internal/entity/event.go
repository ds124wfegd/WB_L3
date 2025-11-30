package entity

import (
	"time"
)

type Event struct {
	ID          int64     `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description" db:"description"`
	Date        time.Time `json:"date" db:"date"`
	TotalSeats  int       `json:"total_seats" db:"total_seats"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type EventWithAvailability struct {
	Event
	AvailableSeats int `json:"available_seats"`
	BookedSeats    int `json:"booked_seats"`
}
