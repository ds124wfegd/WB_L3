package entity

import "time"

type User struct {
	ID         int64     `json:"id" db:"id"`
	Email      string    `json:"email" db:"email"`
	Name       string    `json:"name" db:"name"`
	TelegramID string    `json:"telegram_id" db:"telegram_id"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}
