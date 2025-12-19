package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port              string
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
	DBURL             string
	LogLevel          string
	DBMaxConns        int
	DBMinConns        int
	DBMaxConnIdleTime time.Duration
	DBMaxConnLifetime time.Duration
	MaxRetries        int
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
	if writeTimeoutString := os.Getenv("WRITE_TIMEOUT"); writeTimeoutString != "" {
		if sec, err := strconv.Atoi(writeTimeoutString); err == nil {
			writeTimeout = time.Duration(sec) * time.Second
		}
	}

	idleTimeout := 60 * time.Second // default
	if idleTimeoutString := os.Getenv("IDLE_TIMEOUT"); idleTimeoutString != "" {
		if sec, err := strconv.Atoi(idleTimeoutString); err == nil {
			idleTimeout = time.Duration(sec) * time.Second
		}
	}

	shutdownTimeout := 10 * time.Second // default
	if shutdownTimeoutString := os.Getenv("SHUTDOWN_TIMEOUT"); shutdownTimeoutString != "" {
		if sec, err := strconv.Atoi(shutdownTimeoutString); err == nil {
			shutdownTimeout = time.Duration(sec) * time.Second
		}
	}

	minConnsStr := os.Getenv("DB_MIN_CONNS")
	minConns := 4 // default
	if minConnsStr != "" {
		if v, err := strconv.Atoi(minConnsStr); err == nil {
			minConns = v
		}
	}

	maxConnIdleTime := 30 * time.Minute // default
	if maxConnIdleTimeString := os.Getenv("DB_MAX_CONN_IDLE_TIME"); maxConnIdleTimeString != "" {
		if min, err := strconv.Atoi(maxConnIdleTimeString); err == nil {
			maxConnIdleTime = time.Duration(min) * time.Minute
		}
	}

	maxConnLifetime := 1 * time.Hour // default
	if maxConnLifetimeString := os.Getenv("DB_MAX_CONN_LIFETIME"); maxConnLifetimeString != "" {
		if h, err := strconv.Atoi(maxConnLifetimeString); err == nil {
			maxConnLifetime = time.Duration(h) * time.Hour
		}
	}

	maxRetriesStr := os.Getenv("MAX_RETRIES")
	maxRetries := 3 // default
	if maxRetriesStr != "" {
		if v, err := strconv.Atoi(maxRetriesStr); err == nil {
			maxRetries = v
		}
	}

	return &Config{
		Port:            os.Getenv("APP_PORT"),
		LogLevel:        os.Getenv("LOG_LEVEL"),
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		IdleTimeout:     idleTimeout,
		ShutdownTimeout: shutdownTimeout,
		DBURL: fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s",
			os.Getenv("DB_USER"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
		DBMaxConns:        maxConns,
		DBMinConns:        minConns,
		DBMaxConnIdleTime: maxConnIdleTime,
		DBMaxConnLifetime: maxConnLifetime,
		MaxRetries:        maxRetries,
	}, err
}
