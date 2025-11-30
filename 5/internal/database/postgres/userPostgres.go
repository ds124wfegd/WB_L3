package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ds124wfegd/WB_L3/5/internal/entity"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (email, name, telegram_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	return r.db.QueryRowContext(ctx, query,
		user.Email,
		user.Name,
		user.TelegramID,
		user.CreatedAt,
	).Scan(&user.ID)
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*entity.User, error) {
	query := `
		SELECT id, email, name, telegram_id, created_at
		FROM users 
		WHERE id = $1
	`

	var user entity.User
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.TelegramID,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, name, telegram_id, created_at
		FROM users 
		WHERE email = $1
	`

	var user entity.User
	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.TelegramID,
		&user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) GetByTelegramID(ctx context.Context, telegramID string) (*entity.User, error) {
	query := `
		SELECT id, email, name, telegram_id, created_at
		FROM users 
		WHERE telegram_id = $1
	`

	var user entity.User
	err := r.db.QueryRowContext(ctx, query, telegramID).Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.TelegramID,
		&user.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) UpdateTelegramID(ctx context.Context, userID int64, telegramID string) error {
	query := `UPDATE users SET telegram_id = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, telegramID, userID)
	return err
}

func (r *userRepository) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users 
		SET email = $1, name = $2, telegram_id = $3
		WHERE id = $4
	`

	result, err := r.db.ExecContext(ctx, query,
		user.Email,
		user.Name,
		user.TelegramID,
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) Delete(ctx context.Context, id int64) error {
	// Сначала проверяем, есть ли у пользователя активные бронирования
	var activeBookingsCount int
	query := `SELECT COUNT(*) FROM bookings WHERE user_id = $1 AND status IN ('pending', 'confirmed')`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&activeBookingsCount)
	if err != nil {
		return fmt.Errorf("failed to check user bookings: %w", err)
	}

	if activeBookingsCount > 0 {
		return fmt.Errorf("cannot delete user with active bookings")
	}

	// Удаляем пользователя
	query = `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return entity.ErrUserNotFound
	}

	return nil
}

func (r *userRepository) GetAll(ctx context.Context) ([]*entity.User, error) {
	query := `
		SELECT id, email, name, telegram_id, created_at
		FROM users 
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all users: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.TelegramID,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

func (r *userRepository) SearchByName(ctx context.Context, name string) ([]*entity.User, error) {
	query := `
		SELECT id, email, name, telegram_id, created_at
		FROM users 
		WHERE name ILIKE $1
		ORDER BY name ASC
	`

	searchPattern := "%" + name + "%"
	rows, err := r.db.QueryContext(ctx, query, searchPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search users by name: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		var user entity.User
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Name,
			&user.TelegramID,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	return users, nil
}
