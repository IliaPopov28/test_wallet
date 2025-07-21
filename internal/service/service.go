package service

import (
	"context"
	"errors"
	"log/slog"
	"test_wallet/internal/repository"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/shopspring/decimal"
)

//go:generate mockgen -source=service.go -destination=../../test/mock_wallet_repository.go -package=test WalletRepository

type WalletRepository interface {
	UpdateBalance(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal, opType string) (decimal.Decimal, bool, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (decimal.Decimal, error)
}

type WalletService struct {
	repo       WalletRepository
	logger     *slog.Logger
	maxRetries int
}

func NewWalletService(repo WalletRepository, logger *slog.Logger) *WalletService {
	return &WalletService{
		repo:       repo,
		logger:     logger,
		maxRetries: 3, // можно вынести в .env
	}
}

func (s *WalletService) Deposit(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) (decimal.Decimal, bool, error) {
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		balance, created, err := s.repo.UpdateBalance(ctx, walletID, amount, "DEPOSIT")
		if err == nil {
			return balance, created, nil
		}
		if isRetryableError(err) {
			s.logger.Warn("Retrying deposit",
				slog.String("wallet_id", walletID.String()),
				slog.Int("attempt", i+1),
				slog.Any("err", err),
			)
			time.Sleep(time.Duration(1<<i) * 10 * time.Microsecond)
			lastErr = err
			continue
		}

		if errors.Is(err, repository.ErrWalletNotFound) {
			s.logger.Error("Deposit failed: wallet not found",
				slog.String("wallet_id", walletID.String()),
				slog.Any("amount", amount),
			)
			return balance, false, repository.ErrWalletNotFound
		}
		if errors.Is(err, repository.ErrInsufficientFunds) {
			s.logger.Error("Deposit failed: insufficient funds (should not happen for deposit)",
				slog.String("wallet_id", walletID.String()),
				slog.Any("amount", amount),
				slog.Any("balance", balance),
			)
			return balance, false, repository.ErrInsufficientFunds
		}
		s.logger.Error("Deposit failed: unknown error",
			slog.String("wallet_id", walletID.String()),
			slog.Any("amount", amount),
			slog.Any("err", err),
		)
		return balance, created, err
	}
	s.logger.Error("Deposit failed after retries",
		slog.String("wallet_id", walletID.String()),
		slog.Any("amount", amount),
		slog.Any("err", lastErr),
	)
	return decimal.Zero, false, lastErr
}

func (s *WalletService) Withdraw(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) (decimal.Decimal, error) {
	if amount.IsZero() || amount.IsNegative() {
		s.logger.Error("Withdraw failed: amount must be positive",
			slog.String("wallet_id", walletID.String()),
			slog.Any("amount", amount),
		)
		return decimal.Zero, repository.ErrInvalidAmount
	}
	var lastErr error
	for i := 0; i < s.maxRetries; i++ {
		balance, _, err := s.repo.UpdateBalance(ctx, walletID, amount.Neg(), "WITHDRAW")
		if err == nil {
			return balance, nil
		}
		if isRetryableError(err) {
			s.logger.Warn("Retrying withdraw",
				slog.String("wallet_id", walletID.String()),
				slog.Int("attempt", i+1),
				slog.Any("err", err),
			)
			time.Sleep(time.Duration(1<<i) * 10 * time.Microsecond)
			lastErr = err
			continue
		}

		if errors.Is(err, repository.ErrWalletNotFound) {
			s.logger.Error("Withdraw failed: wallet not found",
				slog.String("wallet_id", walletID.String()),
				slog.Any("amount", amount),
			)
			return balance, repository.ErrWalletNotFound
		}
		if errors.Is(err, repository.ErrInsufficientFunds) {
			s.logger.Warn("Withdraw failed: insufficient funds",
				slog.String("wallet_id", walletID.String()),
				slog.Any("amount", amount),
				slog.Any("balance", balance),
			)
			return balance, repository.ErrInsufficientFunds
		}
		s.logger.Error("Withdraw failed: unknown error",
			slog.String("wallet_id", walletID.String()),
			slog.Any("amount", amount),
			slog.Any("err", err),
		)
		return balance, err
	}
	s.logger.Error("Withdraw failed after retries",
		slog.String("wallet_id", walletID.String()),
		slog.Any("amount", amount),
		slog.Any("err", lastErr),
	)
	return decimal.Zero, lastErr
}

func (s *WalletService) GetBalance(ctx context.Context, walletID uuid.UUID) (decimal.Decimal, error) {
	balance, err := s.repo.GetBalance(ctx, walletID)
	if err != nil {
		if errors.Is(err, repository.ErrWalletNotFound) {
			s.logger.Warn("GetBalance: wallet not found",
				slog.String("wallet_id", walletID.String()),
			)
			return balance, repository.ErrWalletNotFound
		}
		s.logger.Error("GetBalance failed",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return balance, err
	}
	// Не логируем успешное получение баланса
	return balance, nil
}

func isRetryableError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "40001" || pgErr.Code == "40P01"
	}
	return false
}
