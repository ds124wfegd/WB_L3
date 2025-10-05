CREATE TABLE IF NOT EXISTS urls (
    id VARCHAR(36) PRIMARY KEY,
    original_url TEXT NOT NULL,
    short_url VARCHAR(50) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    clicks INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS clicks (
    id VARCHAR(36) PRIMARY KEY,
    short_url VARCHAR(50) NOT NULL,
    user_agent TEXT,
    ip_address VARCHAR(45),
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (short_url) REFERENCES urls(short_url) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_urls_short_url ON urls(short_url);
CREATE INDEX IF NOT EXISTS idx_clicks_short_url ON clicks(short_url);
CREATE INDEX IF NOT EXISTS idx_clicks_timestamp ON clicks(timestamp);