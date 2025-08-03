package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

// NewRouter создает и настраивает маршрутизатор для приложения.
func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(h.LoggingMiddleware)
	r.Use(h.RecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))

	// POST /api/send - выполнение денежного перевода
	r.Post("/api/send", h.HandleSend)

	// GET /api/transactions?count=N - получение последних транзакций
	r.Get("/api/transactions", h.HandleGetLastTransactions)

	// GET /api/wallet/{address}/balance - получение баланса кошелька
	r.Get("/api/wallet/{address}/balance", h.HandleGetBalance)

	return r
}
