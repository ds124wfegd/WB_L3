package postgres

import (
	"database/sql"

	"github.com/ds124wfegd/WB_L3/2/internal/entity"

	_ "github.com/lib/pq"
)

type URLRepository struct {
	db *sql.DB
}

func NewURLRepository(db *sql.DB) URLRepositoryInterface {
	return &URLRepository{db: db}
}

func (r *URLRepository) Create(url *entity.URL) error {
	query := `INSERT INTO urls (id, original_url, short_url, created_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.Exec(query, url.ID, url.OriginalURL, url.ShortURL, url.CreatedAt)
	return err
}

func (r *URLRepository) GetByShortURL(shortURL string) (*entity.URL, error) {
	var url entity.URL
	query := `SELECT id, original_url, short_url, created_at, clicks FROM urls WHERE short_url = $1`
	err := r.db.QueryRow(query, shortURL).Scan(&url.ID, &url.OriginalURL, &url.ShortURL, &url.CreatedAt, &url.Clicks)
	if err != nil {
		return nil, err
	}
	return &url, nil
}

func (r *URLRepository) Exists(shortURL string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM urls WHERE short_url = $1`
	err := r.db.QueryRow(query, shortURL).Scan(&count)
	return count > 0, err
}

func (r *URLRepository) GetAll() ([]entity.URL, error) {
	query := `SELECT id, original_url, short_url, created_at, clicks FROM urls ORDER BY created_at DESC`
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var urls []entity.URL
	for rows.Next() {
		var url entity.URL
		err := rows.Scan(&url.ID, &url.OriginalURL, &url.ShortURL, &url.CreatedAt, &url.Clicks)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}

	return urls, nil
}

func (r *URLRepository) IncrementClicks(shortURL string) error {
	query := `UPDATE urls SET clicks = clicks + 1 WHERE short_url = $1`
	_, err := r.db.Exec(query, shortURL)
	return err
}
