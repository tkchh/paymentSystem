// Пакет models содержит структуры данных приложения
package models

type Wallet struct {
	Address string `json:"address"`
	Balance int    `json:"balance"`
}

type Transaction struct {
	From      string  `json:"from"`
	To        string  `json:"to"`
	Amount    float64 `json:"amount"`
	Timestamp string  `json:"timestamp"`
}
