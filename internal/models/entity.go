package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Wallet struct {
	ID      uuid.UUID       `db:"id" json:"walletId"`
	Balance decimal.Decimal `db:"balance" json:"balance"`
}

type Transaction struct {
	ID        int64           `db:"id"`
	WalletID  uuid.UUID       `db:"wallet_id"`
	Type      string          `db:"type"`
	Amount    decimal.Decimal `db:"amount"`
	CreatedAt time.Time       `db:"created_at"`
}
