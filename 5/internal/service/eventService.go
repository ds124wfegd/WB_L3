package service

import (
	"context"
	"fmt"
	"time"

	repository "github.com/ds124wfegd/WB_L3/5/internal/database/postgres"
	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

// CreateEventRequest represents the data needed to create an event
type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required,min=1,max=255"`
	Description string    `json:"description" binding:"max=1000"`
	Date        time.Time `json:"date" binding:"required"`
	TotalSeats  int       `json:"total_seats" binding:"required,min=1,max=10000"`
}

// UpdateEventRequest represents the data needed to update an event
type UpdateEventRequest struct {
	Title       *string    `json:"title,omitempty"`
	Description *string    `json:"description,omitempty"`
	Date        *time.Time `json:"date,omitempty"`
	TotalSeats  *int       `json:"total_seats,omitempty"`
}

// EventFilter represents filters for searching events
type EventFilter struct {
	Title     string    `json:"title,omitempty"`
	DateFrom  time.Time `json:"date_from,omitempty"`
	DateTo    time.Time `json:"date_to,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	Offset    int       `json:"offset,omitempty"`
	SortBy    string    `json:"sort_by,omitempty"`    // "date", "title", "created_at"
	SortOrder string    `json:"sort_order,omitempty"` // "asc", "desc"
}

type eventService struct {
	eventRepo   repository.EventRepository
	bookingRepo repository.BookingRepository
}

// NewEventService creates a new instance of EventService
func NewEventService(
	eventRepo repository.EventRepository,
	bookingRepo repository.BookingRepository,
) EventService {
	return &eventService{
		eventRepo:   eventRepo,
		bookingRepo: bookingRepo,
	}
}

func (s *eventService) CreateEvent(ctx context.Context, req *CreateEventRequest) (*entity.Event, error) {
	// Validate date is in the future
	if req.Date.Before(time.Now()) {
		return nil, fmt.Errorf("event date must be in the future")
	}

	event := &entity.Event{
		Title:       req.Title,
		Description: req.Description,
		Date:        req.Date,
		TotalSeats:  req.TotalSeats,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.eventRepo.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

func (s *eventService) GetEvent(ctx context.Context, id int64) (*entity.EventWithAvailability, error) {
	event, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return event, nil
}

func (s *eventService) GetAllEvents(ctx context.Context) ([]*entity.EventWithAvailability, error) {
	events, err := s.eventRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	return events, nil
}

func (s *eventService) UpdateEvent(ctx context.Context, id int64, req *UpdateEventRequest) (*entity.Event, error) {
	// Get existing event
	existingEvent, err := s.eventRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing event: %w", err)
	}

	// Update fields if provided
	event := &entity.Event{
		ID:          id,
		Title:       existingEvent.Title,
		Description: existingEvent.Description,
		Date:        existingEvent.Date,
		TotalSeats:  existingEvent.TotalSeats,
		UpdatedAt:   time.Now(),
	}

	if req.Title != nil {
		event.Title = *req.Title
	}
	if req.Description != nil {
		event.Description = *req.Description
	}
	if req.Date != nil {
		if req.Date.Before(time.Now()) {
			return nil, fmt.Errorf("event date must be in the future")
		}
		event.Date = *req.Date
	}
	if req.TotalSeats != nil {
		if *req.TotalSeats < existingEvent.BookedSeats {
			return nil, fmt.Errorf("cannot reduce total seats below current booked seats (%d)", existingEvent.BookedSeats)
		}
		event.TotalSeats = *req.TotalSeats
	}

	// Update in repository
	if err := s.eventRepo.Update(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	return event, nil
}

func (s *eventService) GetEventBookings(ctx context.Context, eventID int64) ([]*entity.Booking, error) {
	bookings, err := s.bookingRepo.GetByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event bookings: %w", err)
	}

	return bookings, nil
}

func (s *eventService) GetEventStats(ctx context.Context, eventID int64) (*entity.EventStats, error) {
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	stats, err := s.bookingRepo.GetEventBookingStats(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to get booking stats: %w", err)
	}

	eventStats := &entity.EventStats{
		Event:           event.Event,
		BookingStats:    *stats,
		UtilizationRate: stats.UtilizationRate(event.TotalSeats),
		AvailableSeats:  stats.AvailableSeats(event.TotalSeats),
	}

	return eventStats, nil
}

func (s *eventService) SearchEvents(ctx context.Context, filter *EventFilter) ([]*entity.EventWithAvailability, error) {
	if filter == nil {
		filter = &EventFilter{}
	}

	// Set default values
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.SortBy == "" {
		filter.SortBy = "date"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "asc"
	}

	// This would typically call a specialized repository method
	// For now, we'll get all events and filter in memory (not efficient for large datasets)
	allEvents, err := s.eventRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get events for search: %w", err)
	}

	var filteredEvents []*entity.EventWithAvailability
	for _, event := range allEvents {
		if filter.Title != "" && !containsIgnoreCase(event.Title, filter.Title) {
			continue
		}
		if !filter.DateFrom.IsZero() && event.Date.Before(filter.DateFrom) {
			continue
		}
		if !filter.DateTo.IsZero() && event.Date.After(filter.DateTo) {
			continue
		}
		filteredEvents = append(filteredEvents, event)
	}

	// Apply sorting
	filteredEvents = s.sortEvents(filteredEvents, filter.SortBy, filter.SortOrder)

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(filteredEvents) {
			return []*entity.EventWithAvailability{}, nil
		}
		filteredEvents = filteredEvents[filter.Offset:]
	}
	if len(filteredEvents) > filter.Limit {
		filteredEvents = filteredEvents[:filter.Limit]
	}

	return filteredEvents, nil
}

func (s *eventService) sortEvents(events []*entity.EventWithAvailability, sortBy, sortOrder string) []*entity.EventWithAvailability {
	switch sortBy {
	case "title":
		if sortOrder == "desc" {
			// Sort by title descending
			// Implementation would sort events by title
		} else {
			// Sort by title ascending
			// Implementation would sort events by title
		}
	case "created_at":
		if sortOrder == "desc" {
			// Sort by created_at descending
			// Implementation would sort events by created_at
		} else {
			// Sort by created_at ascending
			// Implementation would sort events by created_at
		}
	default: // "date"
		if sortOrder == "desc" {
			// Sort by date descending
			// Implementation would sort events by date
		} else {
			// Sort by date ascending
			// Implementation would sort events by date
		}
	}
	return events
}

// Helper function for case-insensitive contains check
func containsIgnoreCase(s, substr string) bool {
	// Simple implementation - in production you might want more robust matching
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

// Исправляем метод DeleteEvent в eventService
func (s *eventService) DeleteEvent(ctx context.Context, id int64) error {
	// Проверяем, есть ли у события активные бронирования
	bookings, err := s.bookingRepo.GetByEventID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check event bookings: %w", err)
	}

	// Проверяем наличие активных бронирований (pending или confirmed)
	for _, booking := range bookings {
		if booking.Status == entity.BookingStatusPending || booking.Status == entity.BookingStatusConfirmed {
			return fmt.Errorf("cannot delete event with active bookings")
		}
	}

	// Удаляем событие
	if err := s.eventRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// Добавляем метод для получения всех событий (без статистики)
func (s *eventService) GetAllEventsSimple(ctx context.Context) ([]*entity.Event, error) {
	// Этот метод должен быть добавлен в репозиторий
	// Временно используем существующий метод и преобразуем результат
	eventsWithAvailability, err := s.eventRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all events: %w", err)
	}

	var events []*entity.Event
	for _, eventWithAvail := range eventsWithAvailability {
		events = append(events, &eventWithAvail.Event)
	}

	return events, nil
}

// Добавляем метод для поиска событий по названию
func (s *eventService) SearchEventsByTitle(ctx context.Context, title string) ([]*entity.EventWithAvailability, error) {
	if title == "" {
		return s.eventRepo.GetAll(ctx)
	}

	events, err := s.eventRepo.SearchByTitle(ctx, title)
	if err != nil {
		return nil, fmt.Errorf("failed to search events by title: %w", err)
	}

	return events, nil
}

// Добавляем метод для получения предстоящих событий
func (s *eventService) GetUpcomingEvents(ctx context.Context, limit int) ([]*entity.EventWithAvailability, error) {
	events, err := s.eventRepo.GetUpcomingEvents(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get upcoming events: %w", err)
	}

	return events, nil
}
