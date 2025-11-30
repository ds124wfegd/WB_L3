package postgres

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/ds124wfegd/WB_L3/5/config"

	_ "github.com/lib/pq"
)

func NewPostgresDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
	return db, nil
}

func RunMigrations(db *sql.DB) error {
	// Read migration files and execute them
	// This is a simplified version - you might want to use a proper migration tool
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS events (
			id SERIAL PRIMARY KEY,
			title VARCHAR(255) NOT NULL,
			description TEXT,
			date TIMESTAMP NOT NULL,
			total_seats INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(255) NOT NULL,
			telegram_id VARCHAR(100),
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS bookings (
			id SERIAL PRIMARY KEY,
			event_id INTEGER REFERENCES events(id),
			user_id INTEGER REFERENCES users(id),
			seats INTEGER NOT NULL,
			status VARCHAR(20) DEFAULT 'pending',
			expires_at TIMESTAMP NOT NULL,
			reservation_timeout INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,

		// Indexes
		`CREATE INDEX IF NOT EXISTS idx_bookings_event_id ON bookings(event_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_user_id ON bookings(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_status ON bookings(status)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_expires_at ON bookings(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_bookings_event_status ON bookings(event_id, status)`,
	}

	for _, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("failed to execute migration: %v", err)
		}
	}

	log.Println("Database migrations completed successfully")
	return nil
}
