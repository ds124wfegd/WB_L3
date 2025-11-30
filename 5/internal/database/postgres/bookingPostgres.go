package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

type bookingRepository struct {
	db *sql.DB
}

func NewBookingRepository(db *sql.DB) BookingRepository {
	return &bookingRepository{db: db}
}

// Create creates a new booking with transaction to ensure data consistency
func (r *bookingRepository) Create(ctx context.Context, booking *entity.Booking) error {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check available seats
	var confirmedSeats int
	query := `SELECT COALESCE(SUM(seats), 0) FROM bookings WHERE event_id = $1 AND status = 'confirmed'`
	err = tx.QueryRowContext(ctx, query, booking.EventID).Scan(&confirmedSeats)
	if err != nil {
		return fmt.Errorf("failed to check confirmed seats: %v", err)
	}

	var totalSeats int
	query = `SELECT total_seats FROM events WHERE id = $1`
	err = tx.QueryRowContext(ctx, query, booking.EventID).Scan(&totalSeats)
	if err != nil {
		return fmt.Errorf("failed to get event total seats: %v", err)
	}

	// Check if user already has a pending or confirmed booking for this event
	var existingBookingCount int
	query = `SELECT COUNT(*) FROM bookings WHERE event_id = $1 AND user_id = $2 AND status IN ('pending', 'confirmed')`
	err = tx.QueryRowContext(ctx, query, booking.EventID, booking.UserID).Scan(&existingBookingCount)
	if err != nil {
		return fmt.Errorf("failed to check existing bookings: %v", err)
	}
	if existingBookingCount > 0 {
		return fmt.Errorf("user already has a booking for this event")
	}

	// Validate available seats
	if confirmedSeats+booking.Seats > totalSeats {
		return fmt.Errorf("not enough available seats: requested %d, available %d",
			booking.Seats, totalSeats-confirmedSeats)
	}

	// Create booking
	query = `
		INSERT INTO bookings (
			event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	expiresAt := now.Add(time.Duration(booking.ReservationTimeout) * time.Minute)

	err = tx.QueryRowContext(ctx, query,
		booking.EventID,
		booking.UserID,
		booking.Seats,
		booking.Status,
		expiresAt,
		booking.ReservationTimeout,
		now,
		now,
	).Scan(&booking.ID)

	if err != nil {
		return fmt.Errorf("failed to create booking: %v", err)
	}

	booking.ExpiresAt = expiresAt
	booking.CreatedAt = now
	booking.UpdatedAt = now

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// GetByID retrieves a booking by its ID
func (r *bookingRepository) GetByID(ctx context.Context, id int64) (*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE id = $1
	`

	var booking entity.Booking
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&booking.ID,
		&booking.EventID,
		&booking.UserID,
		&booking.Seats,
		&booking.Status,
		&booking.ExpiresAt,
		&booking.ReservationTimeout,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, entity.ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking: %v", err)
	}

	return &booking, nil
}

// GetByEventAndUser retrieves a booking by event and user
func (r *bookingRepository) GetByEventAndUser(ctx context.Context, eventID, userID int64) (*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE event_id = $1 AND user_id = $2 AND status IN ('pending', 'confirmed')
		ORDER BY created_at DESC
		LIMIT 1
	`

	var booking entity.Booking
	err := r.db.QueryRowContext(ctx, query, eventID, userID).Scan(
		&booking.ID,
		&booking.EventID,
		&booking.UserID,
		&booking.Seats,
		&booking.Status,
		&booking.ExpiresAt,
		&booking.ReservationTimeout,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking by event and user: %v", err)
	}

	return &booking, nil
}

// UpdateStatus updates the status of a booking
func (r *bookingRepository) UpdateStatus(ctx context.Context, id int64, status entity.BookingStatus) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get current booking to validate the update
	var currentBooking entity.Booking
	query := `SELECT event_id, seats, status FROM bookings WHERE id = $1`
	err = tx.QueryRowContext(ctx, query, id).Scan(
		&currentBooking.EventID,
		&currentBooking.Seats,
		&currentBooking.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to get current booking: %v", err)
	}

	// If changing from pending to confirmed, check seat availability
	if currentBooking.Status == entity.BookingStatusPending && status == entity.BookingStatusConfirmed {
		var confirmedSeats int
		query = `SELECT COALESCE(SUM(seats), 0) FROM bookings WHERE event_id = $1 AND status = 'confirmed'`
		err = tx.QueryRowContext(ctx, query, currentBooking.EventID).Scan(&confirmedSeats)
		if err != nil {
			return fmt.Errorf("failed to check confirmed seats: %v", err)
		}

		var totalSeats int
		query = `SELECT total_seats FROM events WHERE id = $1`
		err = tx.QueryRowContext(ctx, query, currentBooking.EventID).Scan(&totalSeats)
		if err != nil {
			return fmt.Errorf("failed to get event total seats: %v", err)
		}

		if confirmedSeats+currentBooking.Seats > totalSeats {
			return fmt.Errorf("not enough available seats to confirm booking")
		}
	}

	// Update the status
	query = `UPDATE bookings SET status = $1, updated_at = $2 WHERE id = $3`
	result, err := tx.ExecContext(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update booking status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}
	if rowsAffected == 0 {
		return entity.ErrBookingNotFound
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// GetByEventID retrieves all bookings for a specific event
func (r *bookingRepository) GetByEventID(ctx context.Context, eventID int64) ([]*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE event_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings by event: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %v", err)
	}

	return bookings, nil
}

// GetByUserID retrieves all bookings for a specific user
func (r *bookingRepository) GetByUserID(ctx context.Context, userID int64) ([]*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings by user: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %v", err)
	}

	return bookings, nil
}

// GetByStatus retrieves all bookings with a specific status
func (r *bookingRepository) GetByStatus(ctx context.Context, status entity.BookingStatus) ([]*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE status = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings by status: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %v", err)
	}

	return bookings, nil
}

// GetByEventAndStatus retrieves bookings for a specific event and status
func (r *bookingRepository) GetByEventAndStatus(ctx context.Context, eventID int64, status entity.BookingStatus) ([]*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE event_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, eventID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings by event and status: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %v", err)
	}

	return bookings, nil
}

// GetExpiredBookings retrieves expired bookings before a certain time
func (r *bookingRepository) GetExpiredBookings(ctx context.Context, before time.Time) ([]*entity.BookingExpiration, error) {
	query := `
		SELECT 
			b.id, b.expires_at, b.user_id, b.event_id,
			u.telegram_id, u.name as user_name,
			e.title as event_title
		FROM bookings b
		JOIN users u ON b.user_id = u.id
		JOIN events e ON b.event_id = e.id
		WHERE b.status = 'pending' AND b.expires_at < $1
		ORDER BY b.expires_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired bookings: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.BookingExpiration
	for rows.Next() {
		var booking entity.BookingExpiration
		err := rows.Scan(
			&booking.BookingID,
			&booking.ExpiresAt,
			&booking.UserID,
			&booking.EventID,
			&booking.TelegramID,
			&booking.UserName,
			&booking.EventTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expired booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expired bookings: %v", err)
	}

	return bookings, nil
}

// GetExpiringBookings retrieves bookings that will expire within a time range
func (r *bookingRepository) GetExpiringBookings(ctx context.Context, from, to time.Time) ([]*entity.BookingExpiration, error) {
	query := `
		SELECT 
			b.id, b.expires_at, b.user_id, b.event_id,
			u.telegram_id, u.name as user_name,
			e.title as event_title
		FROM bookings b
		JOIN users u ON b.user_id = u.id
		JOIN events e ON b.event_id = e.id
		WHERE b.status = 'pending' AND b.expires_at BETWEEN $1 AND $2
		ORDER BY b.expires_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed to query expiring bookings: %v", err)
	}
	defer rows.Close()

	var bookings []*entity.BookingExpiration
	for rows.Next() {
		var booking entity.BookingExpiration
		err := rows.Scan(
			&booking.BookingID,
			&booking.ExpiresAt,
			&booking.UserID,
			&booking.EventID,
			&booking.TelegramID,
			&booking.UserName,
			&booking.EventTitle,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan expiring booking: %v", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating expiring bookings: %v", err)
	}

	return bookings, nil
}

// DeleteExpired deletes expired bookings and returns the count of deleted rows
func (r *bookingRepository) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	query := `DELETE FROM bookings WHERE status = 'pending' AND expires_at < $1`
	result, err := r.db.ExecContext(ctx, query, before)
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired bookings: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %v", err)
	}

	return rowsAffected, nil
}

// BulkUpdateStatus updates the status of multiple bookings in a single transaction
func (r *bookingRepository) BulkUpdateStatus(ctx context.Context, ids []int64, status entity.BookingStatus) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Build the query with placeholders
	query := `UPDATE bookings SET status = $1, updated_at = $2 WHERE id IN (`
	args := []interface{}{status, time.Now()}

	for i, id := range ids {
		if i > 0 {
			query += ","
		}
		query += fmt.Sprintf("$%d", i+3)
		args = append(args, id)
	}
	query += ")"

	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk update booking status: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected != int64(len(ids)) {
		return fmt.Errorf("expected to update %d rows, but updated %d", len(ids), rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	return nil
}

// CountByEvent counts all bookings for a specific event
func (r *bookingRepository) CountByEvent(ctx context.Context, eventID int64) (int, error) {
	query := `SELECT COUNT(*) FROM bookings WHERE event_id = $1`
	var count int
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings by event: %v", err)
	}
	return count, nil
}

// CountByEventAndStatus counts bookings for a specific event and status
func (r *bookingRepository) CountByEventAndStatus(ctx context.Context, eventID int64, status entity.BookingStatus) (int, error) {
	query := `SELECT COUNT(*) FROM bookings WHERE event_id = $1 AND status = $2`
	var count int
	err := r.db.QueryRowContext(ctx, query, eventID, status).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count bookings by event and status: %v", err)
	}
	return count, nil
}

// GetEventBookingStats returns statistics for event bookings
func (r *bookingRepository) GetEventBookingStats(ctx context.Context, eventID int64) (*entity.EventBookingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_bookings,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN seats ELSE 0 END), 0) as pending_seats,
			COALESCE(SUM(CASE WHEN status = 'confirmed' THEN seats ELSE 0 END), 0) as confirmed_seats,
			COALESCE(SUM(CASE WHEN status = 'cancelled' THEN seats ELSE 0 END), 0) as cancelled_seats,
			COALESCE(SUM(CASE WHEN status = 'expired' THEN seats ELSE 0 END), 0) as expired_seats
		FROM bookings 
		WHERE event_id = $1
	`

	var stats entity.EventBookingStats
	err := r.db.QueryRowContext(ctx, query, eventID).Scan(
		&stats.TotalBookings,
		&stats.PendingSeats,
		&stats.ConfirmedSeats,
		&stats.CancelledSeats,
		&stats.ExpiredSeats,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get event booking stats: %v", err)
	}

	return &stats, nil
}

// LockBooking locks a booking for update (for concurrency control)
func (r *bookingRepository) LockBooking(ctx context.Context, id int64) error {
	query := `SELECT 1 FROM bookings WHERE id = $1 FOR UPDATE`
	var dummy int
	err := r.db.QueryRowContext(ctx, query, id).Scan(&dummy)
	if err != nil {
		return fmt.Errorf("failed to lock booking: %v", err)
	}
	return nil
}

// GetWithLock retrieves a booking with a lock for update
func (r *bookingRepository) GetWithLock(ctx context.Context, id int64) (*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		WHERE id = $1
		FOR UPDATE
	`

	var booking entity.Booking
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&booking.ID,
		&booking.EventID,
		&booking.UserID,
		&booking.Seats,
		&booking.Status,
		&booking.ExpiresAt,
		&booking.ReservationTimeout,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, entity.ErrBookingNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get booking with lock: %v", err)
	}

	return &booking, nil
}

func (r *bookingRepository) Update(ctx context.Context, booking *entity.Booking) error {
	query := `
		UPDATE bookings 
		SET event_id = $1, user_id = $2, seats = $3, status = $4, 
		    expires_at = $5, reservation_timeout = $6, updated_at = $7
		WHERE id = $8
	`

	result, err := r.db.ExecContext(ctx, query,
		booking.EventID,
		booking.UserID,
		booking.Seats,
		booking.Status,
		booking.ExpiresAt,
		booking.ReservationTimeout,
		time.Now(),
		booking.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrBookingNotFound
	}

	booking.UpdatedAt = time.Now()
	return nil
}

func (r *bookingRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM bookings WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete booking: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrBookingNotFound
	}

	return nil
}

func (r *bookingRepository) GetAll(ctx context.Context) ([]*entity.Booking, error) {
	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating bookings: %w", err)
	}

	return bookings, nil
}

func (r *bookingRepository) GetRecentBookings(ctx context.Context, limit int) ([]*entity.Booking, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT 
			id, event_id, user_id, seats, status, expires_at, 
			reservation_timeout, created_at, updated_at
		FROM bookings 
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*entity.Booking
	for rows.Next() {
		var booking entity.Booking
		err := rows.Scan(
			&booking.ID,
			&booking.EventID,
			&booking.UserID,
			&booking.Seats,
			&booking.Status,
			&booking.ExpiresAt,
			&booking.ReservationTimeout,
			&booking.CreatedAt,
			&booking.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan booking: %w", err)
		}
		bookings = append(bookings, &booking)
	}

	return bookings, nil
}
