package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"log/slog"
	"paymentSystem/internal/models"
	"paymentSystem/internal/storage"
	"time"
)

type Storage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewStorage(db *sql.DB, logger *slog.Logger) *Storage {
	return &Storage{db: db, logger: logger}
}

func (s *Storage) Init() error {
	if _, err := s.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("PRAGMA foreign_keys = ON: %v", err)
	}

	if err := s.createTables(); err != nil {
		return fmt.Errorf("create tables: %v", err)
	}
	return s.seedWallets()
}

func (s *Storage) createTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS wallets (
		    address TEXT NOT NULL PRIMARY KEY,
		    balance REAL NOT NULL
		);

		CREATE TABLE IF NOT EXISTS transactions (
		    id INTEGER PRIMARY KEY AUTOINCREMENT,
		    from_address TEXT NOT NULL,
		    to_address TEXT NOT NULL,
		    amount REAL NOT NULL,
		    created_at DATETIME CURRENT_TIMESTAMP,
		    FOREIGN KEY (from_address) REFERENCES wallets(address),
		    FOREIGN KEY (to_address) REFERENCES wallets(address)
		);
	`)
	return err
}

func (s *Storage) seedWallets() error {
	const count = 10
	var existing int
	err := s.db.QueryRow("SELECT COUNT(*) FROM wallets").Scan(&existing)
	if err != nil {
		return fmt.Errorf("failet to check wallets: %v", err)
	}

	if existing > 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failet to begin transaction: %v", err)
	}
	defer tx.Rollback()

	for i := 1; i <= count; i++ {
		address := uuid.NewString()
		_, err = tx.Exec("INSERT INTO wallets (address, balance) VALUES (?, ?)", address, 100)
		if err != nil {
			return fmt.Errorf("failet to insert wallet %d: %v", i, err)
		}
	}

	return tx.Commit()
}

func (s *Storage) Transfer(from, to string, amount float64) error {
	if amount <= 0 {
		return storage.ErrIncorrectAmount
	}
	if from == to {
		return errors.New("cannot transfer to yourself")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failet to begin transaction: %v", err)
	}
	defer tx.Rollback()

	//ПРОВЕРКА КОШЕЛЬКОВ
	var balance float64
	err = tx.QueryRow("SELECT balance FROM wallets WHERE address = ?", from).Scan(&balance)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("sender not found")
		}
		return err
	}
	if balance < amount {
		return storage.ErrInsufficientMoney
	}
	var toExists bool
	err = tx.QueryRow("Select 1 FROM wallets WHERE address = ?", to).Scan(&toExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("receiver not found")
		}
		return err
	}
	//^ПРОВЕРКА КОШЕЛЬКОВ

	_, err = tx.Exec("UPDATE wallets SET balance = balance + ? WHERE address = ?", amount, to)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance - ? WHERE address = ?", amount, from)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO transactions (from_address, to_address, amount, created_at) VALUES (?, ?, ?, ?)", from, to, amount, time.Now())

	return tx.Commit()
}

func (s *Storage) GetBalance(address string) (float64, error) {
	var balance float64
	err := s.db.QueryRow("SELECT balance FROM wallets WHERE address = ?", address).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return balance, storage.ErrWalletNotFound
	}
	return balance, err
}

func (s *Storage) GetLastNTransactions(n int) ([]models.Transaction, error) {
	var transactions []models.Transaction
	rows, err := s.db.Query(`
		SELECT from_address, to_address, amount,created_at 
		FROM transactions
		ORDER BY created_at DESC
		LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tx models.Transaction
		if err = rows.Scan(&tx.From, &tx.To, &tx.Amount, &tx.Timestamp); err != nil {
			return transactions, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}
