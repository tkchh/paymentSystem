package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"paymentSystem/internal/models"
	"testing"

	"paymentSystem/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockService реализует интерфейс services.TransactionService
type mockService struct {
	mock.Mock
}

func (m *mockService) MakeTransaction(from, to string, amount float64) error {
	args := m.Called(from, to, amount)
	return args.Error(0)
}

func (m *mockService) GetBalance(address string) (float64, error) {
	args := m.Called(address)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockService) GetRecentTransactions(n int) ([]models.Transaction, error) {
	args := m.Called(n)
	return args.Get(0).([]models.Transaction), args.Error(1)
}

// setupTestHandler создаёт обработчик с мок-сервисом
func setupTestHandler() (*Handler, *mockService) {
	mockSvc := new(mockService)
	handler := NewHandler(mockSvc, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return handler, mockSvc
}

func TestHandleSend_Success(t *testing.T) {
	handler, mockSvc := setupTestHandler()

	// Настраиваем мок
	mockSvc.On("MakeTransaction", "wallet-01", "wallet-02", 10.0).Return(nil)

	// Формируем запрос
	reqBody := `{"from": "wallet-01", "to": "wallet-02", "amount": 10.0}`
	req := httptest.NewRequest("POST", "/api/send", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Выполняем запрос
	handler.HandleSend(w, req)

	// Проверяем результат
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status": "success"}`, w.Body.String())
	mockSvc.AssertExpectations(t)
}

func TestHandleSend_InvalidJSON(t *testing.T) {
	handler, _ := setupTestHandler()

	// Некорректный JSON (amount как строка)
	reqBody := `{"from": "wallet-01", "to": "wallet-02", "amount": "ten"}`
	req := httptest.NewRequest("POST", "/api/send", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSend(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestHandleSend_InsufficientFunds(t *testing.T) {
	handler, mockSvc := setupTestHandler()

	// Настраиваем мок для возврата ошибки
	mockSvc.On("MakeTransaction", "wallet-01", "wallet-02", 50.0).Return(storage.ErrInsufficientFunds)

	reqBody := `{"from": "wallet-01", "to": "wallet-02", "amount": 50.0}`
	req := httptest.NewRequest("POST", "/api/send", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleSend(w, req)

	assert.Equal(t, http.StatusPaymentRequired, w.Code)
	assert.JSONEq(t, `{"error": "insufficient funds"}`, w.Body.String())
}

func TestHandleGetBalance_Success(t *testing.T) {
	handler, mockSvc := setupTestHandler()

	// Настраиваем мок
	mockSvc.On("GetBalance", "wallet-01").Return(100.0, nil)

	// Создаем запрос с параметром
	req := httptest.NewRequest("GET", "/api/wallet/wallet-01/balance", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("address", "wallet-01")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.HandleGetBalance(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"balance": 100}`, w.Body.String())
}

func TestHandleGetBalance_NotFound(t *testing.T) {
	handler, mockSvc := setupTestHandler()

	// Настраиваем мок
	mockSvc.On("GetBalance", "invalid-wallet").Return(0.0, storage.ErrWalletNotFound)

	req := httptest.NewRequest("GET", "/api/wallet/invalid-wallet/balance", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("address", "invalid-wallet")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	w := httptest.NewRecorder()

	handler.HandleGetBalance(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.JSONEq(t, `{"error": "wallet not found"}`, w.Body.String())
}

func TestHandleGetLastTransactions_Success(t *testing.T) {
	handler, mockSvc := setupTestHandler()

	// Настраиваем мок
	transactions := []models.Transaction{
		{From: "wallet-01", To: "wallet-02", Amount: 10.0},
	}
	mockSvc.On("GetRecentTransactions", 5).Return(transactions, nil)

	req := httptest.NewRequest("GET", "/api/transactions?count=5", nil)
	w := httptest.NewRecorder()

	handler.HandleGetLastTransactions(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	expected, _ := json.Marshal(transactions)
	assert.JSONEq(t, string(expected), w.Body.String())
}

func TestHandleGetLastTransactions_InvalidCount(t *testing.T) {
	handler, _ := setupTestHandler()

	req := httptest.NewRequest("GET", "/api/transactions?count=abc", nil)
	w := httptest.NewRecorder()

	handler.HandleGetLastTransactions(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.JSONEq(t, `{"error": "invalid count"}`, w.Body.String())
}
