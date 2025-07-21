package testutil

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// SetupTestDB запускает контейнер Postgres, ждёт его готовности, применяет миграции и возвращает пул и функцию очистки.
func SetupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()
	postgresC, err := tcpostgres.Run(ctx,
		"postgres:17-alpine",
		tcpostgres.WithDatabase("wallets"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("secret"),
	)
	assert.NoError(t, err)

	dbURL, err := postgresC.ConnectionString(ctx, "sslmode=disable")
	assert.NoError(t, err)

	var pool *pgxpool.Pool
	for i := 0; i < 20; i++ {
		pool, err = pgxpool.New(ctx, dbURL)
		if err == nil {
			err = pool.Ping(ctx)
			if err == nil {
				break
			}
			pool.Close()
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		// Если не удалось подключиться — вывести логи контейнера
		fmt.Fprintln(os.Stderr, "[testutil] Postgres did not become ready in time. Container logs:")
		logs, logErr := postgresC.Logs(ctx)
		if logErr == nil {
			io.Copy(os.Stderr, logs)
		} else {
			fmt.Fprintln(os.Stderr, "[testutil] Failed to get container logs:", logErr)
		}
	}
	assert.NoError(t, err, "Postgres did not become ready in time")

	// Миграции
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS wallets (
			id UUID PRIMARY KEY,
			balance DECIMAL(15, 2) NOT NULL DEFAULT 0 CHECK (balance >= 0)
		);
		CREATE TABLE IF NOT EXISTS transactions (
			id SERIAL PRIMARY KEY,
			wallet_id UUID NOT NULL REFERENCES wallets(id),
			type VARCHAR(10) NOT NULL CHECK (type IN ('DEPOSIT', 'WITHDRAW')),
			amount DECIMAL(15, 2) NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);
		CREATE INDEX IF NOT EXISTS idx_transactions_wallet ON transactions(wallet_id);
	`)
	assert.NoError(t, err)

	return pool, func() {
		pool.Close()
		postgresC.Terminate(ctx)
	}
}
