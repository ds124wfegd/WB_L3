// Ininicializing common application configuration
package config

import (
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	App      AppConfig      `mapstructure:"app"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Email    EmailConfig    `mapstructure:"email"`
	Telegram TelegramConfig `mapstructure:"telegram"`
	Booking  BookingConfig  `mapstructure:"booking"`
	Worker   WorkerConfig   `mapstructure:"worker"`
	Redis    RedisConfig    `mapstructure:"redis"`
}

type ServerConfig struct {
	AppVersion   string `json:"appVersion"`
	Host         string `json:"host" validate:"required"`
	Port         string `json:"port" validate:"required"`
	Timeout      time.Duration
	Idle_timeout time.Duration
	Env          string `json:"environment"`
	Mode         string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	DBName          string        `mapstructure:"dbname"`
	SSLMode         string        `mapstructure:"sslmode"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
}

type AppConfig struct {
	ShortURLLength int           `mapstructure:"short_url_length"`
	CacheTTL       time.Duration `mapstructure:"cache_ttl"`
	BaseURL        string        `mapstructure:"base_url"`
}

type JWTConfig struct {
	Secret     string        `mapstructure:"secret"`
	Expiration time.Duration `mapstructure:"expiration"`
}

type EmailConfig struct {
	From     string `mapstructure:"from"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	Enabled  bool   `mapstructure:"enabled"`
}

type TelegramConfig struct {
	BotToken string `mapstructure:"bot_token"`
	ChatID   string `mapstructure:"chat_id"`
	Enabled  bool   `mapstructure:"enabled"`
}

type BookingConfig struct {
	DefaultTimeout int `mapstructure:"default_timeout"` // в минутах
	MaxSeats       int `mapstructure:"max_seats"`
}

type WorkerConfig struct {
	CleanupInterval int `mapstructure:"cleanup_interval"` // в минутах
	BatchSize       int `mapstructure:"batch_size"`
}

type RedisConfig struct {
	URL      string `json:"URL"`
	Host     string `json:"host" validate:"required"`
	Port     int    `json:"port" validate:"required"`
	Password string `json:"password" validate:"required"`
	DB       int    `json:"db" validate:"required"`

	// Настройки пула соединений
	MaxRetries   int
	PoolSize     int
	MinIdleConns int
	MaxIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolTimeout  time.Duration
}

func LoadConfig() (*viper.Viper, error) {

	viperInstance := viper.New()

	viperInstance.AddConfigPath("./config")
	viperInstance.SetConfigName("config")
	viperInstance.SetConfigType("yaml")

	err := viperInstance.ReadInConfig()

	if err != nil {
		return nil, err
	}
	return viperInstance, nil
}

func ParseConfig(v *viper.Viper) (*Config, error) {

	var c Config

	err := v.Unmarshal(&c)
	if err != nil {
		log.Fatalf("unable to decode config into struct, %v", err)
		return nil, err
	}
	return &c, nil
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

/*


// GetServerAddress возвращает полный адрес сервера
func (c *Config) GetServerAddress() string {
	return c.Server.Host + ":" + c.Server.Port
}

// IsProduction проверяет, production ли окружение
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// IsDevelopment проверяет, development ли окружение
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// GetDatabaseURL возвращает DSN строку для подключения к БД
func (c *Config) GetDatabaseURL() string {
	return "postgres://" + c.Database.User + ":" + c.Database.Password +
		"@" + c.Database.Host + ":" + strconv.Itoa(c.Database.Port) +
		"/" + c.Database.DBName + "?sslmode=" + c.Database.SSLMode
}

// GetEmailConfig возвращает конфигурацию email
func (c *Config) GetEmailConfig() *EmailConfig {
	return &c.Email
}

// GetTelegramConfig возвращает конфигурацию Telegram
func (c *Config) GetTelegramConfig() *TelegramConfig {
	return &c.Telegram
}


// setDefaults устанавливает значения по умолчанию
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.appVersion", "1.0.0")
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", "8080")
	v.SetDefault("server.timeout", 30*time.Second)
	v.SetDefault("server.idle_timeout", 60*time.Second)
	v.SetDefault("server.environment", "development")
	v.SetDefault("server.mode", "debug")

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "eventbooker_user")
	v.SetDefault("database.password", "password")
	v.SetDefault("database.dbname", "eventbooker")
	v.SetDefault("database.sslmode", "disable")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 5)
	v.SetDefault("database.conn_max_lifetime", 5*time.Minute)

	// App defaults
	v.SetDefault("app.short_url_length", 8)
	v.SetDefault("app.cache_ttl", 15*time.Minute)
	v.SetDefault("app.base_url", "http://localhost:8080")

	// JWT defaults
	v.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-in-production")
	v.SetDefault("jwt.expiration", 24*time.Hour)

	// Email defaults
	v.SetDefault("email.from", "noreply@eventbooker.com")
	v.SetDefault("email.host", "smtp.gmail.com")
	v.SetDefault("email.port", 587)
	v.SetDefault("email.enabled", false)

	// Telegram defaults
	v.SetDefault("telegram.enabled", false)

	// Booking defaults
	v.SetDefault("booking.default_timeout", 30) // 30 минут
	v.SetDefault("booking.max_seats", 1000)

	// Worker defaults
	v.SetDefault("worker.cleanup_interval", 1) // 1 минута
	v.SetDefault("worker.batch_size", 100)
}

// GetEnv получает переменную окружения с fallback значением
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvInt получает int переменную окружения с fallback значением
func GetEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// GetEnvBool получает bool переменную окружения с fallback значением
func GetEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// GetEnvDuration получает duration переменную окружения с fallback значением
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
*/
