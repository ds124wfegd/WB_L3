package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

type eventRepository struct {
	db *sql.DB
}

func NewEventRepository(db *sql.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, event *entity.Event) error {
	query := `
		INSERT INTO events (title, description, date, total_seats, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`

	return r.db.QueryRowContext(ctx, query,
		event.Title,
		event.Description,
		event.Date,
		event.TotalSeats,
		time.Now(),
		time.Now(),
	).Scan(&event.ID)
}

func (r *eventRepository) GetByID(ctx context.Context, id int64) (*entity.EventWithAvailability, error) {
	query := `
		SELECT 
			e.id, e.title, e.description, e.date, e.total_seats, e.created_at, e.updated_at,
			COALESCE(SUM(CASE WHEN b.status = 'confirmed' THEN b.seats ELSE 0 END), 0) as booked_seats
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id
		WHERE e.id = $1
		GROUP BY e.id
	`

	var event entity.EventWithAvailability
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&event.ID,
		&event.Title,
		&event.Description,
		&event.Date,
		&event.TotalSeats,
		&event.CreatedAt,
		&event.UpdatedAt,
		&event.BookedSeats,
	)

	if err != nil {
		return nil, err
	}

	event.AvailableSeats = event.TotalSeats - event.BookedSeats
	return &event, nil
}

func (r *eventRepository) GetAll(ctx context.Context) ([]*entity.EventWithAvailability, error) {
	query := `
		SELECT 
			e.id, e.title, e.description, e.date, e.total_seats, e.created_at, e.updated_at,
			COALESCE(SUM(CASE WHEN b.status = 'confirmed' THEN b.seats ELSE 0 END), 0) as booked_seats
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id
		GROUP BY e.id
		ORDER BY e.date
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*entity.EventWithAvailability
	for rows.Next() {
		var event entity.EventWithAvailability
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Description,
			&event.Date,
			&event.TotalSeats,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.BookedSeats,
		)
		if err != nil {
			return nil, err
		}
		event.AvailableSeats = event.TotalSeats - event.BookedSeats
		events = append(events, &event)
	}

	return events, nil
}

func (r *eventRepository) UpdateSeats(ctx context.Context, eventID int64, seats int) error {
	query := `UPDATE events SET total_seats = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, seats, time.Now(), eventID)
	return err
}

func (r *eventRepository) Update(ctx context.Context, event *entity.Event) error {
	query := `
		UPDATE events 
		SET title = $1, description = $2, date = $3, total_seats = $4, updated_at = $5
		WHERE id = $6
	`

	result, err := r.db.ExecContext(ctx, query,
		event.Title,
		event.Description,
		event.Date,
		event.TotalSeats,
		time.Now(),
		event.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrEventNotFound
	}

	return nil
}

func (r *eventRepository) Delete(ctx context.Context, id int64) error {
	// Сначала проверяем, есть ли у события бронирования
	var bookingCount int
	query := `SELECT COUNT(*) FROM bookings WHERE event_id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&bookingCount)
	if err != nil {
		return fmt.Errorf("failed to check event bookings: %w", err)
	}

	if bookingCount > 0 {
		return fmt.Errorf("cannot delete event with existing bookings")
	}

	// Удаляем событие
	query = `DELETE FROM events WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrEventNotFound
	}

	return nil
}

func (r *eventRepository) GetUpcomingEvents(ctx context.Context, limit int) ([]*entity.EventWithAvailability, error) {
	if limit <= 0 {
		limit = 10
	}

	query := `
		SELECT 
			e.id, e.title, e.description, e.date, e.total_seats, e.created_at, e.updated_at,
			COALESCE(SUM(CASE WHEN b.status = 'confirmed' THEN b.seats ELSE 0 END), 0) as booked_seats
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id
		WHERE e.date > $1
		GROUP BY e.id
		ORDER BY e.date ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query upcoming events: %w", err)
	}
	defer rows.Close()

	var events []*entity.EventWithAvailability
	for rows.Next() {
		var event entity.EventWithAvailability
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Description,
			&event.Date,
			&event.TotalSeats,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.BookedSeats,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		event.AvailableSeats = event.TotalSeats - event.BookedSeats
		events = append(events, &event)
	}

	return events, nil
}

func (r *eventRepository) SearchByTitle(ctx context.Context, title string) ([]*entity.EventWithAvailability, error) {
	query := `
		SELECT 
			e.id, e.title, e.description, e.date, e.total_seats, e.created_at, e.updated_at,
			COALESCE(SUM(CASE WHEN b.status = 'confirmed' THEN b.seats ELSE 0 END), 0) as booked_seats
		FROM events e
		LEFT JOIN bookings b ON e.id = b.event_id
		WHERE e.title ILIKE $1
		GROUP BY e.id
		ORDER BY e.date ASC
	`

	searchPattern := "%" + title + "%"
	rows, err := r.db.QueryContext(ctx, query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search events by title: %w", err)
	}
	defer rows.Close()

	var events []*entity.EventWithAvailability
	for rows.Next() {
		var event entity.EventWithAvailability
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Description,
			&event.Date,
			&event.TotalSeats,
			&event.CreatedAt,
			&event.UpdatedAt,
			&event.BookedSeats,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		event.AvailableSeats = event.TotalSeats - event.BookedSeats
		events = append(events, &event)
	}

	return events, nil
}

func (r *eventRepository) GetEventsByDateRange(ctx context.Context, from, to time.Time) ([]*entity.Event, error) {
	query := `
		SELECT id, title, description, date, total_seats, created_at, updated_at
		FROM events
		WHERE date BETWEEN $1 AND $2
		ORDER BY date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query events by date range: %w", err)
	}
	defer rows.Close()

	var events []*entity.Event
	for rows.Next() {
		var event entity.Event
		err := rows.Scan(
			&event.ID,
			&event.Title,
			&event.Description,
			&event.Date,
			&event.TotalSeats,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	return events, nil
}
