package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"test_wallet/internal/repository"
	"test_wallet/internal/service"
	"test_wallet/internal/testutil"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))
var resp struct{ Balance string }

func setupIntegrationRouter(t *testing.T) (*gin.Engine, func()) {
	pool, teardown := testutil.SetupTestDB(t)
	repo := repository.NewWalletPGRepository(pool, testLogger)
	svc := service.NewWalletService(repo, testLogger)
	handler := NewWalletHTTPHandler(svc)
	r := gin.Default()
	handler.RegisterRoutes(r)
	return r, teardown
}

func TestIntegration_Deposit_And_Withdraw(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()

	walletID := uuid.New()
	// DEPOSIT
	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        "100.50",
	})
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	json.Unmarshal(w.Body.Bytes(), &resp)
	d, _ := decimal.NewFromString(resp.Balance)
	assert.True(t, d.Equal(decimal.NewFromFloat(100.5)))

	// DEPOSIT (ещё раз)
	body, _ = json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        "50.25",
	})
	req, _ = http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &resp)
	d, _ = decimal.NewFromString(resp.Balance)
	assert.True(t, d.Equal(decimal.NewFromFloat(150.75)))

	// WITHDRAW (успех)
	body, _ = json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "WITHDRAW",
		"amount":        "50.75",
	})
	req, _ = http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	json.Unmarshal(w.Body.Bytes(), &resp)
	d, _ = decimal.NewFromString(resp.Balance)
	assert.True(t, d.Equal(decimal.NewFromInt(100)))

	// WITHDRAW (недостаточно средств)
	body, _ = json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "WITHDRAW",
		"amount":        "200.00",
	})
	req, _ = http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "insufficient funds")
}

func TestIntegration_InvalidAmount(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()
	walletID := uuid.New()
	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        "0",
	})
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "amount must not be zero")
}

func TestIntegration_InvalidUUID(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()
	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      "not-a-uuid",
		"operationType": "DEPOSIT",
		"amount":        "100",
	})
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegration_InvalidOperationType(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()
	walletID := uuid.New()
	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "INVALID_OP",
		"amount":        "100",
	})
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request")
}

func TestIntegration_GetBalance(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()
	walletID := uuid.New()
	// Сначала DEPOSIT
	body, _ := json.Marshal(map[string]interface{}{
		"walletId":      walletID,
		"operationType": "DEPOSIT",
		"amount":        "123.45",
	})
	req, _ := http.NewRequest("POST", "/api/v1/wallet", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)
	// Теперь GET
	req, _ = http.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "123.45")
}

func TestIntegration_GetBalance_NotFound(t *testing.T) {
	r, teardown := setupIntegrationRouter(t)
	defer teardown()
	walletID := uuid.New()
	req, _ := http.NewRequest("GET", "/api/v1/wallets/"+walletID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "wallet not found")
}
