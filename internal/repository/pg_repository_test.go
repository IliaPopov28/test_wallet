package repository_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"test_wallet/internal/repository"
	"test_wallet/internal/testutil"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestUpdateBalance_EdgeCases(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()

	// Попытка снять с несуществующего кошелька
	_, _, err := repo.UpdateBalance(context.Background(), walletID, decimal.NewFromInt(-10), "WITHDRAW")
	assert.ErrorIs(t, err, repository.ErrWalletNotFound)

	// Депозит (создание)
	balance, created, err := repo.UpdateBalance(context.Background(), walletID, decimal.NewFromFloat(100.99), "DEPOSIT")
	assert.NoError(t, err)
	assert.True(t, created)
	assert.True(t, balance.Equal(decimal.NewFromFloat(100.99)))

	// Снятие больше, чем есть
	_, _, err = repo.UpdateBalance(context.Background(), walletID, decimal.NewFromFloat(-200), "WITHDRAW")
	assert.ErrorIs(t, err, repository.ErrInsufficientFunds)
}

func TestUpdateBalance_ConcurrentDeposits(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()
	_ = repo.CreateWallet(context.Background(), walletID)

	var wg sync.WaitGroup
	for i := 0; i < 2000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := repo.UpdateBalance(context.Background(), walletID, decimal.NewFromInt(1), "DEPOSIT")
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	balance, err := repo.GetBalance(context.Background(), walletID)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(2000)))
}

func TestUpdateBalance_ZeroAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()
	amount := decimal.Zero
	_, _, err := repo.UpdateBalance(context.Background(), walletID, amount, "DEPOSIT")
	assert.Error(t, err)
}

func TestUpdateBalance_NegativeAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(-100)
	_, _, err := repo.UpdateBalance(context.Background(), walletID, amount, "DEPOSIT")
	assert.Error(t, err)
}

func TestUpdateBalance_BigAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1e12)
	balance, created, err := repo.UpdateBalance(context.Background(), walletID, amount, "DEPOSIT")
	assert.NoError(t, err)
	assert.True(t, created)
	assert.True(t, balance.Equal(amount))
}

func TestGetBalance_NotFound(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	walletID := uuid.New()
	balance, err := repo.GetBalance(context.Background(), walletID)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
}
