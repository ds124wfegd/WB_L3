package entity

import (
	"time"
)

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	SendTime  time.Time `json:"send_time"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Attempts  int       `json:"attempts"`
}

type NotificationRequest struct {
	UserID   string    `json:"user_id" binding:"required"`
	Title    string    `json:"title" binding:"required"`
	Message  string    `json:"message" binding:"required"`
	SendTime time.Time `json:"send_time" binding:"required"`
}

const (
	StatusPending   = "pending"
	StatusSent      = "sent"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)
