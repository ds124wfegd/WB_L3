package entity

import "time"

type ShortenRequest struct {
	URL         string `json:"url" binding:"required"`
	CustomShort string `json:"custom_short,omitempty"`
}

type URL struct {
	ID          string    `json:"id"`
	OriginalURL string    `json:"original_url"`
	ShortURL    string    `json:"short_url"`
	CreatedAt   time.Time `json:"created_at"`
	Clicks      int       `json:"clicks"`
}

type Click struct {
	ID        string    `json:"id"`
	ShortURL  string    `json:"short_url"`
	UserAgent string    `json:"user_agent"`
	IPAddress string    `json:"ip_address"`
	Timestamp time.Time `json:"timestamp"`
}

type Analytics struct {
	TotalClicks int             `json:"total_clicks"`
	DailyStats  []DailyStat     `json:"daily_stats"`
	UserAgents  []UserAgentStat `json:"user_agents"`
}

type DailyStat struct {
	Date   string `json:"date"`
	Clicks int    `json:"clicks"`
}

type UserAgentStat struct {
	UserAgent string `json:"user_agent"`
	Clicks    int    `json:"clicks"`
}

type ShortenResponse struct {
	ShortURL     string    `json:"short_url"`
	OriginalURL  string    `json:"original_url"`
	CreatedAt    time.Time `json:"created_at"`
	ShortURLFull string    `json:"short_url_full"`
}
