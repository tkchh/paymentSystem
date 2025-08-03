// Пакет sqlite содержит реализацию интерфейса storage.Storage для SQLite
// Реализует:
// - Инициализацию базы данных и создание таблиц
// - Сидирование тестовых кошельков при первом запуске
// - Операции с кошельками и транзакциями
//
// Использует транзакции для обеспечения целостности данных,
// особенно при выполнении денежных переводов.
package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log/slog"
	"paymentSystem/internal/models"
	"paymentSystem/internal/storage"
	"strconv"
	"time"
)

type Storage struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewStorage(db *sql.DB, logger *slog.Logger) *Storage {
	return &Storage{db: db, logger: logger}
}

// Init инициализирует базу данных при первом запуске.
func (s *Storage) Init() error {
	if _, err := s.db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("PRAGMA foreign_keys = ON: %v", err)
	}

	if err := s.createTables(); err != nil {
		return fmt.Errorf("create tables: %v", err)
	}
	return s.seedWallets()
}

// createTables создает необходимые таблицы в базе данных.
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

// seedWallets добавляет тестовые кошельки при первом запуске.
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
		address := "wallet-" + strconv.Itoa(i)
		_, err = tx.Exec("INSERT INTO wallets (address, balance) VALUES (?, ?)", address, 100)
		if err != nil {
			return fmt.Errorf("failet to insert wallet %d: %v", i, err)
		}
	}

	return tx.Commit()
}

// Transfer выполняет денежный перевод между кошельками.
func (s *Storage) Transfer(from, to string, amount float64) error {
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
			return storage.ErrWalletNotFound
		}
		return err
	}
	if balance < amount {
		return storage.ErrInsufficientFunds
	}
	var toExists bool
	err = tx.QueryRow("Select 1 FROM wallets WHERE address = ?", to).Scan(&toExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ErrWalletNotFound
		}
		return err
	}
	//^ПРОВЕРКА КОШЕЛЬКОВ

	_, err = tx.Exec("UPDATE wallets SET balance = balance - ? WHERE address = ?", amount, from)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE wallets SET balance = balance + ? WHERE address = ?", amount, to)
	if err != nil {
		return err
	}

	_, err = tx.Exec("INSERT INTO transactions (from_address, to_address, amount, created_at) VALUES (?, ?, ?, ?)", from, to, amount, time.Now())

	return tx.Commit()
}

// GetBalance возвращает текущий баланс кошелька.
func (s *Storage) GetBalance(address string) (float64, error) {
	var balance float64
	err := s.db.QueryRow("SELECT balance FROM wallets WHERE address = ?", address).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return balance, storage.ErrWalletNotFound
	}
	return balance, err
}

// GetLastNTransactions возвращает последние N транзакций.
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
