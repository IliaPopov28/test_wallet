package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type WalletRequest struct {
	WalletID      uuid.UUID       `json:"walletId" binding:"required"`
	OperationType string          `json:"operationType" binding:"required,oneof=DEPOSIT WITHDRAW"`
	Amount        decimal.Decimal `json:"amount" binding:"required"`
}
