package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	DBURL      string
	LogLevel   string
	DBMaxConns int
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load("config.env")
	maxConnsStr := os.Getenv("DB_MAX_CONNS")
	maxConns := 8 // default
	if maxConnsStr != "" {
		if v, err := strconv.Atoi(maxConnsStr); err == nil {
			maxConns = v
		}
	}
	return &Config{
		Port:     os.Getenv("APP_PORT"),
		LogLevel: os.Getenv("LOG_LEVEL"),
		DBURL: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
		DBMaxConns: maxConns,
	}, nil
}
