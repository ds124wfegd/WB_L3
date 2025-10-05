package postgres

import (
	"database/sql"

	"github.com/ds124wfegd/WB_L3/2/internal/entity"
)

type AnalyticsRepository struct {
	db *sql.DB
}

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{db: db}
}

func (r *AnalyticsRepository) RecordClick(click *entity.Click) error {
	query := `INSERT INTO clicks (id, short_url, user_agent, ip_address, timestamp) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.Exec(query, click.ID, click.ShortURL, click.UserAgent, click.IPAddress, click.Timestamp)
	return err
}

func (r *AnalyticsRepository) GetAnalytics(shortURL string) (*entity.Analytics, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM urls WHERE short_url = $1)", shortURL).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, sql.ErrNoRows
	}

	var totalClicks int
	err = r.db.QueryRow("SELECT clicks FROM urls WHERE short_url = $1", shortURL).Scan(&totalClicks)
	if err != nil {
		return nil, err
	}

	dailyQuery := `
        SELECT DATE(timestamp) as date, COUNT(*) as clicks 
        FROM clicks 
        WHERE short_url = $1 
        GROUP BY DATE(timestamp) 
        ORDER BY date DESC
        LIMIT 30
    `
	rows, err := r.db.Query(dailyQuery, shortURL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dailyStats []entity.DailyStat
	for rows.Next() {
		var stat entity.DailyStat
		err := rows.Scan(&stat.Date, &stat.Clicks)
		if err != nil {
			return nil, err
		}
		dailyStats = append(dailyStats, stat)
	}

	uaQuery := `
        SELECT user_agent, COUNT(*) as clicks 
        FROM clicks 
        WHERE short_url = $1 
        GROUP BY user_agent 
        ORDER BY clicks DESC
    `
	uaRows, err := r.db.Query(uaQuery, shortURL)
	if err != nil {
		return nil, err
	}
	defer uaRows.Close()

	var userAgents []entity.UserAgentStat
	for uaRows.Next() {
		var ua entity.UserAgentStat
		err := uaRows.Scan(&ua.UserAgent, &ua.Clicks)
		if err != nil {
			return nil, err
		}
		userAgents = append(userAgents, ua)
	}

	return &entity.Analytics{
		TotalClicks: totalClicks,
		DailyStats:  dailyStats,
		UserAgents:  userAgents,
	}, nil
}
