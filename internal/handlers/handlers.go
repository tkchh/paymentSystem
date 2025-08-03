// Пакет handlers содержит HTTP-обработчики
//
// - Выполнение переводов
// - Просмотр баланса
// - Получение истории переводов
package handlers

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"paymentSystem/internal/services"
	"paymentSystem/internal/storage"
	"strconv"
)

type Handler struct {
	service services.TransactionService
	logger  *slog.Logger
}

func NewHandler(service services.TransactionService, logger *slog.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// respondJSON формирует JSON-ответ с указанным статусом.
func (h *Handler) respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

// respondError формирует стандартный ответ об ошибке.
func (h *Handler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}

// HandleSend обрабатывает запрос на выполнение денежного перевода.
func (h *Handler) HandleSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Amount float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.service.MakeTransaction(req.From, req.To, req.Amount); err != nil {
		h.handleError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// HandleGetBalance обрабатывает запрос на получение баланса кошелька.
func (h *Handler) HandleGetBalance(w http.ResponseWriter, r *http.Request) {
	address := chi.URLParam(r, "address")
	if address == "" {
		h.respondError(w, http.StatusBadRequest, "address is required")
		return
	}

	balance, err := h.service.GetBalance(address)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]float64{"balance": balance})
}

// HandleGetLastTransactions обрабатывает запрос на получение последних транзакций.
func (h *Handler) HandleGetLastTransactions(w http.ResponseWriter, r *http.Request) {
	count := 0
	n := r.URL.Query().Get("count")

	count, err := strconv.Atoi(n)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "invalid count")
		return
	}

	transactions, err := h.service.GetRecentTransactions(count)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.respondJSON(w, http.StatusOK, transactions)
}

// handleError обрабатывает ошибки от сервисного слоя.
func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrInvalidAmount),
		errors.Is(err, services.ErrSelfTransfer):
		h.respondError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, storage.ErrWalletNotFound):
		h.respondError(w, http.StatusNotFound, err.Error())

	case errors.Is(err, storage.ErrInsufficientFunds):
		h.respondError(w, http.StatusPaymentRequired, err.Error())

	default:
		h.logger.Error("internal error", "error", err)
		h.respondError(w, http.StatusInternalServerError, "internal error")
	}
}

// Правила преобразования:
// - Ошибки валидации → 400 Bad Request
// - Кошелек не найден → 404 Not Found
// - Недостаточно средств → 402 Payment Required
// - Все остальные ошибки → 500 Internal Server Error

// Формат запроса:
// {
//   "from": "адрес_отправителя",
//   "to": "адрес_получателя",
//   "amount": число
// }
