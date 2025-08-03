// services/service_test.go
package services

import (
	"bytes"
	"errors"
	"log/slog"
	"paymentSystem/internal/models"
	"paymentSystem/internal/storage"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockStorage реализует интерфейс storage.Storage для тестов
type mockStorage struct {
	transferFn             func(from, to string, amount float64) error
	getBalanceFn           func(address string) (float64, error)
	getLastNTransactionsFn func(n int) ([]models.Transaction, error)
}

func (m *mockStorage) Init() error {
	panic("not implemented")
}

func (m *mockStorage) GetBalance(address string) (float64, error) {
	if m.getBalanceFn != nil {
		return m.getBalanceFn(address)
	}
	panic("not implemented")
}

func (m *mockStorage) Transfer(from, to string, amount float64) error {
	if m.transferFn != nil {
		return m.transferFn(from, to, amount)
	}
	panic("not implemented")
}

func (m *mockStorage) GetLastNTransactions(n int) ([]models.Transaction, error) {
	if m.getLastNTransactionsFn != nil {
		return m.getLastNTransactionsFn(n)
	}
	panic("not implemented")
}

// setupTestService создаёт сервис с моком и тестовым логгером
func setupTestService() (TransactionService, *mockStorage) {
	mock := &mockStorage{}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	service := NewTransactionService(mock, logger)
	return service, mock
}

func TestMakeTransaction_InvalidAmount(t *testing.T) {
	service, _ := setupTestService()

	err := service.MakeTransaction("a", "b", -100)
	assert.ErrorIs(t, err, ErrInvalidAmount)
}

func TestMakeTransaction_SelfTransfer(t *testing.T) {
	service, _ := setupTestService()

	err := service.MakeTransaction("a", "a", 100)
	assert.ErrorIs(t, err, ErrSelfTransfer)
}

func TestMakeTransaction_InsufficientFunds(t *testing.T) {
	service, mock := setupTestService()

	mock.transferFn = func(from, to string, amount float64) error {
		return storage.ErrInsufficientFunds
	}

	err := service.MakeTransaction(uuid.NewString(), uuid.NewString(), 100)
	assert.ErrorIs(t, err, storage.ErrInsufficientFunds)
}

func TestMakeTransaction_WalletNotFound(t *testing.T) {
	service, mock := setupTestService()

	mock.transferFn = func(from, to string, amount float64) error {
		return storage.ErrWalletNotFound
	}

	err := service.MakeTransaction(uuid.NewString(), uuid.NewString(), 100)
	assert.ErrorIs(t, err, storage.ErrWalletNotFound)
}

func TestMakeTransaction_Success(t *testing.T) {
	service, mock := setupTestService()
	validUUID_1 := uuid.NewString()
	validUUID_2 := uuid.NewString()

	mock.transferFn = func(from, to string, amount float64) error {
		assert.Equal(t, validUUID_1, from)
		assert.Equal(t, validUUID_2, to)
		assert.Equal(t, 50.0, amount)
		return nil
	}

	err := service.MakeTransaction(validUUID_1, validUUID_2, 50)
	assert.NoError(t, err)
}

func TestGetBalance_WalletNotFound(t *testing.T) {
	service, mock := setupTestService()
	wallet := uuid.NewString()

	mock.getBalanceFn = func(address string) (float64, error) {
		return 0, storage.ErrWalletNotFound
	}

	_, err := service.GetBalance(wallet)
	assert.ErrorIs(t, err, storage.ErrWalletNotFound)
}

func TestGetBalance_Success(t *testing.T) {
	service, mock := setupTestService()
	validUUID := uuid.NewString()

	mock.getBalanceFn = func(address string) (float64, error) {
		assert.Equal(t, validUUID, address)
		return 100.0, nil
	}

	balance, err := service.GetBalance(validUUID)
	assert.NoError(t, err)
	assert.Equal(t, 100.0, balance)
}

func TestGetRecentTransactions_Error(t *testing.T) {
	service, mock := setupTestService()

	mock.getLastNTransactionsFn = func(n int) ([]models.Transaction, error) {
		return nil, errors.New("db error")
	}

	_, err := service.GetRecentTransactions(10)
	assert.Error(t, err)
}
