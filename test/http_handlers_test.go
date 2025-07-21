package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"test_wallet/internal/handlers"
	"testing"

	"test_wallet/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestHandleWalletOperation_Deposit_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	walletID := uuid.New()
	mockService.EXPECT().
		Deposit(gomock.Any(), walletID, decimal.NewFromInt(100)).
		Return(decimal.NewFromInt(200), false, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        "100",
	})

	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "200")
}

func TestHandleWalletOperation_Withdraw_InsufficientFunds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	walletID := uuid.New()
	mockService.EXPECT().
		Withdraw(gomock.Any(), walletID, decimal.NewFromInt(100)).
		Return(decimal.NewFromInt(0), repository.ErrInsufficientFunds)

	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "WITHDRAW",
		"amount":        "100",
	})

	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "insufficient funds")
}

func TestHandleWalletOperation_InvalidRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	body := []byte(`{"walletId": "not-a-uuid", "operationType": "DEPOSIT", "amount": "100"}`)
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestHandleGetBalance_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	walletID := uuid.New()
	mockService.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(decimal.NewFromInt(500), nil)

	req, _ := http.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "500")
}

func TestHandleGetBalance_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	walletID := uuid.New()
	mockService.EXPECT().
		GetBalance(gomock.Any(), walletID).
		Return(decimal.Zero, repository.ErrWalletNotFound)

	req, _ := http.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "wallet not found")
}

func TestHandleGetBalance_InvalidUUID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockService := NewMockWalletService(ctrl)
	handler := handlers.NewWalletHTTPHandler(mockService)
	r := gin.Default()
	handler.RegisterRoutes(r)

	req, _ := http.NewRequest("GET", "/api/v1/wallets/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid wallet_id")
}
