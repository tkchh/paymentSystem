package storage

import (
	"errors"
	"paymentSystem/internal/models"
)

// Файл возможно избыточен для такого проекта,
// но в случае добавления новой DB легко масштабировать
var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Storage interface {
	Init() error
	GetBalance(address string) (float64, error)
	Transfer(from, to string, amount float64) error
	GetLastNTransactions(n int) ([]models.Transaction, error)
}
