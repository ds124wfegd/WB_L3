// Ininicializing common application configuration
package config

import (
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	App    AppConfig    `mapstructure:"app"`
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

type AppConfig struct {
	ShortURLLength int           `mapstructure:"short_url_length"`
	CacheTTL       time.Duration `mapstructure:"cache_ttl"`
	BaseURL        string        `mapstructure:"base_url"`
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
