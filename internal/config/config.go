package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	DBURL        string
	LogLevel     string
	DBMaxConns   int
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load("config.env")
	maxConnsStr := os.Getenv("DB_MAX_CONNS")
	maxConns := 8 // default
	if maxConnsStr != "" {
		if v, err := strconv.Atoi(maxConnsStr); err == nil {
			maxConns = v
		}
	}

	readTimeout := 5 * time.Second // default
	if readTimeoutString := os.Getenv("READ_TIMEOUT"); readTimeoutString != "" {
		if sec, err := strconv.Atoi(readTimeoutString); err == nil {
			readTimeout = time.Duration(sec) * time.Second
		}
	}

	writeTimeout := 5 * time.Second // default
	if writeTimeoutString := os.Getenv("READ_TIMEOUT"); writeTimeoutString != "" {
		if sec, err := strconv.Atoi(writeTimeoutString); err == nil {
			writeTimeout = time.Duration(sec) * time.Second
		}
	}

	idleTimeout := 60 * time.Second // default
	if idleTimeoutString := os.Getenv("READ_TIMEOUT"); idleTimeoutString != "" {
		if sec, err := strconv.Atoi(idleTimeoutString); err == nil {
			readTimeout = time.Duration(sec) * time.Second
		}
	}

	return &Config{
		Port:         os.Getenv("APP_PORT"),
		LogLevel:     os.Getenv("LOG_LEVEL"),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		DBURL: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
		DBMaxConns: maxConns,
	}, err
}
