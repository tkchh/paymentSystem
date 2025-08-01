package sqlite

import (
	"database/sql"
	"errors"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"log/slog"
	"os"
	"paymentSystem/internal/storage"
	"testing"
	"time"
)

type StorageTestSuite struct {
	suite.Suite
	db      *sql.DB
	storage storage.Storage
	logger  *slog.Logger
}

func (s *StorageTestSuite) SetupTest() {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(s.T(), err)
	s.db = db

	s.logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	s.storage = NewStorage(db, s.logger)
	require.NoError(s.T(), s.storage.Init())
}

func (s *StorageTestSuite) TearDownTest() {
	s.db.Close()
}

func (s *StorageTestSuite) createTestWallet(address string, balance float64) {
	_, err := s.db.Exec("INSERT INTO wallets (address, balance) VALUES (?, ?)", address, balance)
	require.NoError(s.T(), err)
}

func TestStorageSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}

func (s *StorageTestSuite) TestTransfer_Successful() {
	// Arrange
	s.createTestWallet("wallet-1", 100.0)
	s.createTestWallet("wallet-2", 100.0)

	// Act
	err := s.storage.Transfer("wallet-1", "wallet-2", 50.0)

	// Assert
	assert.NoError(s.T(), err)

	balance1, err := s.storage.GetBalance("wallet-1")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 50.0, balance1)

	balance2, err := s.storage.GetBalance("wallet-2")
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 150.0, balance2)

	// Проверяем запись транзакции
	transactions, err := s.storage.GetLastNTransactions(1)
	assert.NoError(s.T(), err)
	require.Len(s.T(), transactions, 1)

	tx := transactions[0]
	assert.Equal(s.T(), "wallet-1", tx.From)
	assert.Equal(s.T(), "wallet-2", tx.To)
	assert.Equal(s.T(), 50.0, tx.Amount)
}

func (s *StorageTestSuite) TestTransfer_InsufficientMoney() {
	// Arrange
	s.createTestWallet("sender", 100.0)
	s.createTestWallet("receiver", 100.0)

	// Act
	err := s.storage.Transfer("sender", "receiver", 150.0)

	// Assert
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "insufficient money")

	// Проверяем, что балансы не изменились
	senderBalance, _ := s.storage.GetBalance("sender")
	assert.Equal(s.T(), 100.0, senderBalance)

	receiverBalance, _ := s.storage.GetBalance("receiver")
	assert.Equal(s.T(), 100.0, receiverBalance)

	// Проверяем отсутствие транзакций
	transactions, _ := s.storage.GetLastNTransactions(10)
	assert.Empty(s.T(), transactions)
}

func (s *StorageTestSuite) TestTransfer_InvalidAmount() {
	testCases := []struct {
		name   string
		amount float64
	}{
		{"Negative amount", -10.0},
		{"Zero amount", 0.0},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Arrange
			s.createTestWallet("wallet-1"+tc.name, 100.0)
			s.createTestWallet("wallet-2"+tc.name, 100.0)

			// Act
			err := s.storage.Transfer("wallet-1", "wallet-2", tc.amount)
			//time.Sleep(1 * time.Second)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "invalid amount")
		})
	}
}

func (s *StorageTestSuite) TestTransfer_NonexistentWallets() {
	testCases := []struct {
		name     string
		from     string
		to       string
		errorMsg string
	}{
		{"Nonexistent sender", "invalid-sender", "valid-receiver", "sender not found"},
		{"Nonexistent receiver", "valid-sender", "invalid-receiver", "receiver not found"},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			// Arrange
			if tc.from == "valid-sender" {
				s.createTestWallet("valid-sender", 100.0)
			}
			if tc.to == "valid-receiver" {
				s.createTestWallet("valid-receiver", 100.0)
			}

			// Act
			err := s.storage.Transfer(tc.from, tc.to, 50.0)

			// Assert
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}

func (s *StorageTestSuite) TestTransfer_SameWallet() {
	// Arrange
	s.createTestWallet("same-wallet", 100.0)

	// Act
	err := s.storage.Transfer("same-wallet", "same-wallet", 50.0)

	// Assert
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot transfer to yourself")

	balance, _ := s.storage.GetBalance("same-wallet")
	assert.Equal(s.T(), 100.0, balance)
}

func (s *StorageTestSuite) TestGetBalance_WalletNotFound() {
	// Act
	balance, err := s.storage.GetBalance("nonexistent-wallet")

	// Assert
	assert.Error(s.T(), err)
	assert.True(s.T(), errors.Is(err, storage.ErrWalletNotFound))
	assert.Zero(s.T(), balance)
}

func (s *StorageTestSuite) TestGetLastNTransactions() {
	// Arrange
	s.createTestWallet("wallet-1", 100.0)
	s.createTestWallet("wallet-2", 100.0)
	s.createTestWallet("wallet-3", 100.0)

	// Выполняем несколько транзакций с задержкой для разных timestamp
	s.Require().NoError(s.storage.Transfer("wallet-1", "wallet-2", 10.0))
	time.Sleep(10 * time.Millisecond)
	s.Require().NoError(s.storage.Transfer("wallet-2", "wallet-3", 20.0))
	time.Sleep(10 * time.Millisecond)
	s.Require().NoError(s.storage.Transfer("wallet-3", "wallet-1", 5.0))

	// Act
	transactions, err := s.storage.GetLastNTransactions(2)

	// Assert
	assert.NoError(s.T(), err)
	require.Len(s.T(), transactions, 2)

	// Проверяем порядок (последние транзакции должны быть первыми)
	assert.Equal(s.T(), "wallet-3", transactions[0].From)
	assert.Equal(s.T(), "wallet-1", transactions[0].To)
	assert.Equal(s.T(), 5.0, transactions[0].Amount)

	assert.Equal(s.T(), "wallet-2", transactions[1].From)
	assert.Equal(s.T(), "wallet-3", transactions[1].To)
	assert.Equal(s.T(), 20.0, transactions[1].Amount)
}

func (s *StorageTestSuite) TestGetLastNTransactions_MoreThanExist() {
	// Arrange
	s.createTestWallet("wallet-1", 100.0)
	s.createTestWallet("wallet-2", 100.0)
	s.Require().NoError(s.storage.Transfer("wallet-1", "wallet-2", 10.0))

	// Act
	transactions, err := s.storage.GetLastNTransactions(10)

	// Assert
	assert.NoError(s.T(), err)
	assert.Len(s.T(), transactions, 1)
}

func (s *StorageTestSuite) TestGetLastNTransactions_Empty() {
	// Act
	transactions, err := s.storage.GetLastNTransactions(5)

	// Assert
	assert.NoError(s.T(), err)
	assert.Empty(s.T(), transactions)
}
