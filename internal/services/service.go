// Пакет services содержит бизнес-логику платежной системы.
//
// Реализует сервисный слой, который:
// - Валидирует входные данные
// - Обрабатывает бизнес-правила
// - Логирует ключевые операции
// - Преобразует ошибки хранилища в понятные бизнес-ошибки
package services

import (
	"errors"
	"log/slog"
	"paymentSystem/internal/models"
	"paymentSystem/internal/storage"
)

// Ошибки, возникающие на уровне сервиса
var (
	// ErrInvalidAmount возвращается при попытке перевода неположительной суммы
	ErrInvalidAmount = errors.New("amount must be positive")

	// ErrSelfTransfer возвращается при попытке перевода самому себе
	ErrSelfTransfer = errors.New("cannot send money to yourself")

	// ErrInternalError возвращается при неожиданных ошибках
	ErrInternalError = errors.New("internal error")
)

type TransactionService interface {
	MakeTransaction(from, to string, amount float64) error
	GetBalance(address string) (float64, error)
	GetRecentTransactions(n int) ([]models.Transaction, error)
}

type transactionService struct {
	storage storage.Storage
	logger  *slog.Logger
}

func NewTransactionService(storage storage.Storage, logger *slog.Logger) TransactionService {
	return &transactionService{
		storage: storage,
		logger:  logger,
	}
}

// MakeTransaction реализует метод интерфейса для выполнения перевода.
func (s *transactionService) MakeTransaction(from, to string, amount float64) error {
	if amount <= 0 {
		s.logger.Warn("invalid amount", "amount", amount)
		return ErrInvalidAmount
	}
	if from == to {
		s.logger.Warn("self transfer attempt", "from", from, "to", to)
		return ErrSelfTransfer
	}

	s.logger.Info("transaction initialized",
		"from", from,
		"to", to,
		"amount", amount,
	)

	if err := s.storage.Transfer(from, to, amount); err != nil {
		return s.handleStorageError(err, amount)
	}

	s.logger.Info("transaction completed",
		"from", from,
		"to", to,
		"amount", amount,
	)

	return nil
}

// GetBalance реализует метод интерфейса для получения баланса.
func (s *transactionService) GetBalance(address string) (float64, error) {
	s.logger.Info("get balance",
		"address", address,
	)

	balance, err := s.storage.GetBalance(address)
	if err != nil {
		s.logger.Error("failed to get balance", "address", address, "error", err)
		return 0, s.handleStorageError(err, 0)
	}

	s.logger.Info("get balance",
		"address", address,
		"balance", balance,
	)

	return balance, nil
}

// GetRecentTransactions реализует метод интерфейса для получения транзакций.
func (s *transactionService) GetRecentTransactions(n int) ([]models.Transaction, error) {
	if n <= 0 {
		s.logger.Warn("invalid amount", "amount", n)
		return nil, ErrInvalidAmount
	}

	s.logger.Info("get recent transactions",
		"count", n,
	)

	transactions, err := s.storage.GetLastNTransactions(n)
	if err != nil {
		return nil, err
	}

	s.logger.Info("get recent transactions",
		"count", n,
		"transactions", transactions,
	)

	return transactions, nil
}

// handleStorageError преобразует ошибки хранилища в бизнес-ошибки.
func (s *transactionService) handleStorageError(err error, amount float64) error {
	switch {
	case errors.Is(err, storage.ErrInsufficientFunds):
		s.logger.Warn("insufficient funds", "amount", amount, "err", err)
		return storage.ErrInsufficientFunds
	case errors.Is(err, storage.ErrWalletNotFound):
		s.logger.Warn("wallet not found", "err", err)
		return storage.ErrWalletNotFound
	default:
		s.logger.Error("unexpected storage error", "err", err)
		return ErrInternalError
	}
}
