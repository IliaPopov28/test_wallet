package handlers

import (
	"context"
	"net/http"
	"test_wallet/internal/models"
	"test_wallet/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

//go:generate mockgen -source=http_handlers.go -destination=../../test/mock_wallet_service.go -package=test WalletService

type WalletService interface {
	Deposit(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) (decimal.Decimal, bool, error)
	Withdraw(ctx context.Context, walletID uuid.UUID, amount decimal.Decimal) (decimal.Decimal, error)
	GetBalance(ctx context.Context, walletID uuid.UUID) (decimal.Decimal, error)
}

type WalletHTTPHandler struct {
	service WalletService
}

func NewWalletHTTPHandler(service WalletService) *WalletHTTPHandler {
	return &WalletHTTPHandler{service: service}
}

func (h *WalletHTTPHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		v1.POST("/wallet", h.HandleWalletOperation)
		v1.GET("/wallets/:wallet_id", h.HandleGetBalance)
	}
}

func (h *WalletHTTPHandler) HandleWalletOperation(c *gin.Context) {
	var req models.WalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request", "details": err.Error()})
		return
	}

	if req.Amount.Cmp(decimal.Zero) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be > 0"})
	}

	switch req.OperationType {
	case "DEPOSIT":
		balance, created, err := h.service.Deposit(c.Request.Context(), req.WalletID, req.Amount)
		if err != nil {
			status := http.StatusServiceUnavailable
			if err == repository.ErrWalletNotFound {
				status = http.StatusNotFound
			} else if err == repository.ErrInsufficientFunds {
				status = http.StatusConflict
			}
			c.JSON(status, gin.H{"error": err.Error(), "balance": balance.String()})
			return
		}
		status := http.StatusOK
		if created {
			status = http.StatusCreated
		}
		c.JSON(status, gin.H{"balance": balance.String()})
	case "WITHDRAW":
		balance, err := h.service.Withdraw(c.Request.Context(), req.WalletID, req.Amount)
		if err != nil {
			status := http.StatusServiceUnavailable
			if err == repository.ErrWalletNotFound {
				status = http.StatusNotFound
			} else if err == repository.ErrInsufficientFunds {
				status = http.StatusConflict
			}
			c.JSON(status, gin.H{"error": err.Error(), "balance": balance.String()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"balance": balance.String()})
	}
}

func (h *WalletHTTPHandler) HandleGetBalance(c *gin.Context) {
	walletIDStr := c.Param("wallet_id")
	walletID, err := uuid.Parse(walletIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid wallet_id"})
		return
	}
	balance, err := h.service.GetBalance(c.Request.Context(), walletID)
	if err != nil {
		status := http.StatusServiceUnavailable
		if err == repository.ErrWalletNotFound {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"balance": balance.String()})
}
