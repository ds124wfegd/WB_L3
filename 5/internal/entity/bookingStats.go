package entity

import (
	"fmt"
	"time"
)

// EventStats содержит комплексную статистику для мероприятия
type EventStats struct {
	Event           Event             `json:"event"`
	BookingStats    EventBookingStats `json:"booking_stats"`
	UtilizationRate float64           `json:"utilization_rate"`
	AvailableSeats  int               `json:"available_seats"`
	PeakBookingTime *time.Time        `json:"peak_booking_time,omitempty"`
	Revenue         float64           `json:"revenue,omitempty"` // Выручка (если мероприятие платное)
	PopularityScore float64           `json:"popularity_score"`  // Оценка популярности 0-100
}

// EventBookingStats содержит статистику бронирований для мероприятия
type EventBookingStats struct {
	TotalBookings  int `json:"total_bookings"`
	PendingSeats   int `json:"pending_seats"`
	ConfirmedSeats int `json:"confirmed_seats"`
	CancelledSeats int `json:"cancelled_seats"`
	ExpiredSeats   int `json:"expired_seats"`
	NoShowSeats    int `json:"no_show_seats"` // Неявки
}

// UserStats содержит статистику пользователя
type UserStats struct {
	User              *User                `json:"user"`
	TotalBookings     int                  `json:"total_bookings"`
	ConfirmedBookings int                  `json:"confirmed_bookings"`
	PendingBookings   int                  `json:"pending_bookings"`
	CancelledBookings int                  `json:"cancelled_bookings"`
	ExpiredBookings   int                  `json:"expired_bookings"`
	FavoriteEvents    []*EventBookingCount `json:"favorite_events"`
	TotalSeatsBooked  int                  `json:"total_seats_booked"`
	AttendanceRate    float64              `json:"attendance_rate"` // Процент посещаемости
	LoyaltyScore      float64              `json:"loyalty_score"`   // Оценка лояльности 0-100
	JoinDate          time.Time            `json:"join_date"`
	LastActivity      *time.Time           `json:"last_activity,omitempty"`
}

// EventBookingCount представляет мероприятие с количеством бронирований
type EventBookingCount struct {
	EventID    int64     `json:"event_id"`
	EventTitle string    `json:"event_title"`
	EventDate  time.Time `json:"event_date"`
	Bookings   int64     `json:"bookings"`
	Seats      int       `json:"seats"`
}

// SystemStats содержит общую статистику системы
type SystemStats struct {
	TotalEvents     int64     `json:"total_events"`
	TotalUsers      int64     `json:"total_users"`
	TotalBookings   int64     `json:"total_bookings"`
	ActiveEvents    int64     `json:"active_events"`    // Активные мероприятия (будущие)
	CompletedEvents int64     `json:"completed_events"` // Завершенные мероприятия
	Revenue         float64   `json:"revenue"`          // Общая выручка
	Utilization     float64   `json:"utilization"`      // Средняя утилизация мест
	PeakUsageTime   time.Time `json:"peak_usage_time"`  // Время пиковой нагрузки
}

// BookingTrends содержит тренды бронирований
type BookingTrends struct {
	Period        string    `json:"period"` // "day", "week", "month"
	Dates         []string  `json:"dates"`
	Bookings      []int64   `json:"bookings"`
	Confirmations []int64   `json:"confirmations"`
	Cancellations []int64   `json:"cancellations"`
	Revenue       []float64 `json:"revenue,omitempty"`
	AverageSeats  []float64 `json:"average_seats"`
}

// AvailableSeats вычисляет доступные места на основе общего количества мест
func (s *EventBookingStats) AvailableSeats(totalSeats int) int {
	return totalSeats - s.ConfirmedSeats
}

// UtilizationRate вычисляет коэффициент утилизации (0.0 до 1.0)
func (s *EventBookingStats) UtilizationRate(totalSeats int) float64 {
	if totalSeats == 0 {
		return 0.0
	}
	return float64(s.ConfirmedSeats) / float64(totalSeats)
}

// CancellationRate вычисляет процент отмен
func (s *EventBookingStats) CancellationRate() float64 {
	totalProcessed := s.ConfirmedSeats + s.CancelledSeats + s.ExpiredSeats
	if totalProcessed == 0 {
		return 0.0
	}
	return float64(s.CancelledSeats+s.ExpiredSeats) / float64(totalProcessed)
}

// ConversionRate вычисляет процент конверсии (подтвержденные / все бронирования)
func (s *EventBookingStats) ConversionRate() float64 {
	totalBookings := s.ConfirmedSeats + s.CancelledSeats + s.ExpiredSeats + s.PendingSeats
	if totalBookings == 0 {
		return 0.0
	}
	return float64(s.ConfirmedSeats) / float64(totalBookings)
}

// CalculatePopularityScore вычисляет оценку популярности мероприятия
func (s *EventStats) CalculatePopularityScore() float64 {
	// Факторы популярности:
	// 1. Утилизация мест (50%)
	// 2. Скорость бронирования (30%)
	// 3. Коэффициент конверсии (20%)

	utilizationScore := s.UtilizationRate * 50

	// Расчет скорости бронирования (места/день)
	daysUntilEvent := s.Event.Date.Sub(time.Now()).Hours() / 24
	if daysUntilEvent <= 0 {
		daysUntilEvent = 1 // Если мероприятие уже прошло
	}
	bookingSpeed := float64(s.BookingStats.ConfirmedSeats) / daysUntilEvent
	maxExpectedSpeed := float64(s.Event.TotalSeats) / 7 // Ожидаемая скорость: все места за неделю
	speedScore := (bookingSpeed / maxExpectedSpeed) * 30
	if speedScore > 30 {
		speedScore = 30
	}

	conversionScore := s.BookingStats.ConversionRate() * 20

	return utilizationScore + speedScore + conversionScore
}

// CalculateAttendanceRate вычисляет процент посещаемости пользователя
func (s *UserStats) CalculateAttendanceRate() float64 {
	totalConfirmed := s.ConfirmedBookings + s.CancelledBookings + s.ExpiredBookings
	if totalConfirmed == 0 {
		return 0.0
	}

	// Предполагаем, что NoShow включены в CancelledBookings
	attended := s.ConfirmedBookings - (s.CancelledBookings / 2) // Упрощенная формула
	return float64(attended) / float64(totalConfirmed) * 100
}

// CalculateLoyaltyScore вычисляет оценку лояльности пользователя
func (s *UserStats) CalculateLoyaltyScore() float64 {
	// Факторы лояльности:
	// 1. Количество подтвержденных бронирований (40%)
	// 2. Процент посещаемости (30%)
	// 3. Активность (давность последней активности) (20%)
	// 4. Разнообразие мероприятий (10%)

	bookingScore := float64(s.ConfirmedBookings) * 2 // Макс 40 очков
	if bookingScore > 40 {
		bookingScore = 40
	}

	attendanceScore := s.AttendanceRate * 0.3 // Макс 30 очков

	activityScore := 0.0
	if s.LastActivity != nil {
		daysSinceActivity := time.Since(*s.LastActivity).Hours() / 24
		if daysSinceActivity <= 7 {
			activityScore = 20
		} else if daysSinceActivity <= 30 {
			activityScore = 15
		} else if daysSinceActivity <= 90 {
			activityScore = 10
		} else {
			activityScore = 5
		}
	}

	diversityScore := float64(len(s.FavoriteEvents)) * 2 // Макс 10 очков
	if diversityScore > 10 {
		diversityScore = 10
	}

	return bookingScore + attendanceScore + activityScore + diversityScore
}

// String возвращает строковое представление статистики мероприятия
func (s *EventStats) String() string {
	return fmt.Sprintf(
		"Event: %s, Utilization: %.1f%%, Available: %d/%d, Popularity: %.1f/100",
		s.Event.Title,
		s.UtilizationRate*100,
		s.AvailableSeats,
		s.Event.TotalSeats,
		s.PopularityScore,
	)
}

// String возвращает строковое представление статистики пользователя
func (s *UserStats) String() string {
	return fmt.Sprintf(
		"User: %s, Bookings: %d, Attendance: %.1f%%, Loyalty: %.1f/100",
		s.User.Name,
		s.TotalBookings,
		s.AttendanceRate,
		s.LoyaltyScore,
	)
}

// IsHighDemand проверяет, есть ли высокий спрос на мероприятие
func (s *EventStats) IsHighDemand() bool {
	return s.UtilizationRate > 0.8 && s.PopularityScore > 70
}

// IsLoyalUser проверяет, является ли пользователь лояльным
func (s *UserStats) IsLoyalUser() bool {
	return s.LoyaltyScore > 70 && s.ConfirmedBookings >= 3
}

// NeedsAttention проверяет, требует ли мероприятие внимания (низкая утилизация)
func (s *EventStats) NeedsAttention() bool {
	daysUntilEvent := s.Event.Date.Sub(time.Now()).Hours() / 24
	return s.UtilizationRate < 0.3 && daysUntilEvent < 7
}
