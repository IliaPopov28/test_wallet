package test

import (
	"context"
	"io"
	"log/slog"
	"test_wallet/internal/repository"
	"test_wallet/internal/service"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

func TestDeposit_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromFloat(100.99)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount, "DEPOSIT").
		Return(decimal.NewFromFloat(100.99), false, nil)

	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromFloat(100.99)))
	assert.False(t, created)
}

func TestDeposit_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount, "DEPOSIT").
		Return(decimal.NewFromInt(0), false, repository.ErrInsufficientFunds)

	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.ErrorIs(t, err, repository.ErrInsufficientFunds)
	assert.True(t, balance.Equal(decimal.NewFromInt(0)))
	assert.False(t, created)
}

func TestDeposit_WalletNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount, "DEPOSIT").
		Return(decimal.NewFromInt(0), false, repository.ErrWalletNotFound)

	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.ErrorIs(t, err, repository.ErrWalletNotFound)
	assert.True(t, balance.Equal(decimal.NewFromInt(0)))
	assert.False(t, created)
}

func TestWithdraw_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(50)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount.Neg(), "WITHDRAW").
		Return(decimal.NewFromInt(50), false, nil)

	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(50)))
}

func TestWithdraw_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount.Neg(), "WITHDRAW").
		Return(decimal.NewFromInt(0), false, repository.ErrInsufficientFunds)

	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.ErrorIs(t, err, repository.ErrInsufficientFunds)
	assert.True(t, balance.Equal(decimal.NewFromInt(0)))
}

func TestWithdraw_WalletNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount.Neg(), "WITHDRAW").
		Return(decimal.NewFromInt(0), false, repository.ErrWalletNotFound)

	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.ErrorIs(t, err, repository.ErrWalletNotFound)
	assert.True(t, balance.Equal(decimal.NewFromInt(0)))
}

func TestGetBalance_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	mockRepo.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(decimal.NewFromInt(200), nil)

	balance, err := svc.GetBalance(context.Background(), walletID)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(200)))
}

func TestGetBalance_WalletNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	mockRepo.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(decimal.Zero, repository.ErrWalletNotFound)

	balance, err := svc.GetBalance(context.Background(), walletID)
	assert.ErrorIs(t, err, repository.ErrWalletNotFound)
	assert.True(t, balance.Equal(decimal.Zero))
}

func TestDeposit_Retry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)
	walletID := uuid.New()
	amount := decimal.NewFromInt(100)
	// Первая попытка — ошибка *pgconn.PgError с кодом 40001 (serialization failure), вторая — успех
	retryErr := &pgconn.PgError{Code: "40001", Message: "serialization failure"}
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount, "DEPOSIT").
		Return(decimal.Zero, false, retryErr).Times(1)
	mockRepo.EXPECT().
		UpdateBalance(gomock.Any(), walletID, amount, "DEPOSIT").
		Return(decimal.NewFromInt(100), false, nil).Times(1)

	balance, created, err := svc.Deposit(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(100)))
	assert.False(t, created)
}

func TestWithdraw_InvalidAmount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)

	walletID := uuid.New()

	balance, err := svc.Withdraw(context.Background(), walletID, decimal.Zero)
	assert.ErrorIs(t, err, repository.ErrInvalidAmount)
	assert.True(t, balance.IsZero())

	balance, err = svc.Withdraw(context.Background(), walletID, decimal.NewFromInt(-10))
	assert.ErrorIs(t, err, repository.ErrInvalidAmount)
	assert.True(t, balance.IsZero())
}

func TestWithdraw_Retry(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)

	walletID := uuid.New()
	amount := decimal.NewFromInt(50)
	negAmount := amount.Neg()

	retryErr := &pgconn.PgError{Code: "40P01", Message: "deadlock detected"}

	gomock.InOrder(
		mockRepo.EXPECT().
			UpdateBalance(gomock.Any(), walletID, negAmount, "WITHDRAW").
			Return(decimal.Zero, false, retryErr),
		mockRepo.EXPECT().
			UpdateBalance(gomock.Any(), walletID, negAmount, "WITHDRAW").
			Return(decimal.NewFromInt(50), false, nil),
	)

	balance, err := svc.Withdraw(context.Background(), walletID, amount)
	assert.NoError(t, err)
	assert.True(t, balance.Equal(decimal.NewFromInt(50)))
}

func TestGetBalance_OtherError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockWalletRepository(ctrl)
	svc := service.NewWalletService(mockRepo, testLogger)

	walletID := uuid.New()
	someError := assert.AnError

	mockRepo.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(decimal.Zero, someError)

	balance, err := svc.GetBalance(context.Background(), walletID)
	assert.ErrorIs(t, err, someError)
	assert.True(t, balance.IsZero())
}
