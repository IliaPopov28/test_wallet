package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrWalletAlreadyExist = errors.New("wallet already exists")
	ErrInvalidAmount      = errors.New("amount must not be zero")
)

type WalletPGRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewWalletPGRepository(pool *pgxpool.Pool, logger *slog.Logger) *WalletPGRepository {
	return &WalletPGRepository{
		pool:   pool,
		logger: logger,
	}
}

func (r *WalletPGRepository) UpdateBalance(
	ctx context.Context,
	walletID uuid.UUID,
	amount decimal.Decimal,
	opType string,
) (decimal.Decimal, bool, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		r.logger.Error("Failed to begin transaction",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return decimal.Zero, false, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			r.logger.Error("Failed to rollback transaction",
				slog.String("wallet_id", walletID.String()),
				slog.Any("err", err),
			)
		}
	}()

	var currentBalance decimal.Decimal
	err = tx.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1 FOR UPDATE", walletID).Scan(&currentBalance)

	if amount.IsZero() {
		return currentBalance, false, ErrInvalidAmount
	}

	created := false
	if err == pgx.ErrNoRows {
		if opType != "DEPOSIT" {
			return decimal.Zero, false, ErrWalletNotFound
		}

		var insertedID uuid.UUID
		err := tx.QueryRow(ctx, `
            INSERT INTO wallets (id, balance) VALUES ($1, 0)
            ON CONFLICT (id) DO NOTHING
            RETURNING id`, walletID).Scan(&insertedID)
		if err != nil && err != pgx.ErrNoRows {
			r.logger.Error("Failed to upsert wallet",
				slog.String("wallet_id", walletID.String()),
				slog.Any("err", err),
			)
			return decimal.Zero, false, err
		}
		if err == nil {
			created = true
		}

		err = tx.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1 FOR UPDATE", walletID).Scan(&currentBalance)
		if err != nil {
			r.logger.Error("Failed to select wallet after upsert",
				slog.String("wallet_id", walletID.String()),
				slog.Any("err", err),
			)
			return decimal.Zero, false, err
		}
	} else if err != nil {
		r.logger.Error("Failed to select wallet for update",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return decimal.Zero, false, err
	}

	newBalance := currentBalance.Add(amount)
	if newBalance.IsNegative() {
		return currentBalance, false, ErrInsufficientFunds
	}

	_, err = tx.Exec(ctx, "UPDATE wallets SET balance = $1 WHERE id = $2", newBalance, walletID)
	if err != nil {
		r.logger.Error("Failed to update wallet balance",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return currentBalance, false, err
	}

	/* _, err = tx.Exec(ctx, "INSERT INTO transactions (wallet_id, type, amount) VALUES ($1, $2, $3)", walletID, opType, amount)
	 if err != nil {
	 	r.logger.Error("Failed to insert transaction",
	 		slog.String("wallet_id", walletID.String()),
	 		slog.String("operation", opType),
	 		slog.Any("amount", amount),
	 		slog.Any("err", err),
	 	)
	 	return currentBalance, false, err
	}*/

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit transaction",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return currentBalance, false, err
	}

	return newBalance, created, nil
}

func (r *WalletPGRepository) GetBalance(ctx context.Context, walletID uuid.UUID) (decimal.Decimal, error) {
	var balance decimal.Decimal
	err := r.pool.QueryRow(ctx, "SELECT balance FROM wallets WHERE id = $1", walletID).Scan(&balance)
	if err == pgx.ErrNoRows {
		return decimal.Zero, ErrWalletNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get balance",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return decimal.Zero, err
	}
	return balance, nil
}

// Для тестов
func (r *WalletPGRepository) CreateWallet(ctx context.Context, walletID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "INSERT INTO wallets (id, balance) VALUES ($1, 0)", walletID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrWalletAlreadyExist
		}
		r.logger.Error("Failed to create wallet",
			slog.String("wallet_id", walletID.String()),
			slog.Any("err", err),
		)
		return err
	}
	return nil
}
