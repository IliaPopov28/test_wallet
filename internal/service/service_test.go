package service_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"

	"test_wallet/internal/repository"
	"test_wallet/internal/service"
	"test_wallet/internal/testutil"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestService_Deposit_Integration(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromFloat(123.45)

	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(amount))
	assert.True(t, created)

	balance, created, err = svc.Deposit(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(amount.Add(amount)))
	assert.False(t, created)
}

func TestService_Withdraw_Integration(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	_, _, _ = svc.Deposit(context.Background(), walletID, amount)

	// WITHDRAW (успех)
	balance, err := svc.Withdraw(context.Background(), walletID, decimal.NewFromInt(50))
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(50)))

	// WITHDRAW (недостаточно средств)
	balance, err = svc.Withdraw(context.Background(), walletID, decimal.NewFromInt(100))
	assert.ErrorIs(t, err, repository.ErrInsufficientFunds)
	assert.True(t, balance.Equal(decimal.NewFromInt(50)))
}

func TestService_ConcurrentDeposits(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	_ = repo.CreateWallet(context.Background(), walletID)

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := svc.Deposit(context.Background(), walletID, decimal.NewFromInt(1))
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	balance, err := svc.GetBalance(context.Background(), walletID)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(1000)))
}

func TestService_Deposit_ZeroAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.Zero
	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
	assert.False(t, created)
}

func TestService_Deposit_NegativeAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(-100)
	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
	assert.False(t, created)
}

func TestService_Deposit_BigAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1e12)
	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(amount))
	assert.True(t, created)
}

func TestService_Withdraw_ZeroAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.Zero
	_, _, _ = svc.Deposit(context.Background(), walletID, decimal.NewFromInt(100))
	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
}

func TestService_Withdraw_NegativeAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(-100)
	_, _, _ = svc.Deposit(context.Background(), walletID, decimal.NewFromInt(100))
	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
}

func TestService_Withdraw_BigAmount(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromFloat(1e12)
	_, _, _ = svc.Deposit(context.Background(), walletID, amount)
	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
}

func TestService_GetBalance_NotFound(t *testing.T) {
	pool, teardown := testutil.SetupTestDB(t)
	defer teardown()
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	walletID := uuid.New()
	balance, err := svc.GetBalance(context.Background(), walletID)
	assert.Error(t, err)
	assert.True(t, balance.Equal(decimal.Zero))
}
