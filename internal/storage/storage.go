package storage

import (
	"errors"
	"paymentSystem/internal/models"
)

var (
	ErrWalletNotFound    = errors.New("wallet not found")
	ErrInsufficientMoney = errors.New("insufficient money")
	ErrIncorrectAmount   = errors.New("invalid amount")
)

type Storage interface {
	Init() error
	GetBalance(address string) (float64, error)
	Transfer(from, to string, amount float64) error
	GetLastNTransactions(n int) ([]models.Transaction, error)
}
