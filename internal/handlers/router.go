package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"time"
)

func NewRouter(h *Handler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(h.LoggingMiddleware)
	r.Use(h.RecoverMiddleware)
	r.Use(middleware.Timeout(60 * time.Second))

	r.Post("/api/send", h.HandleSend)
	r.Get("/api/transactions", h.HandleGetLastTransactions)
	r.Get("/api/wallet/{address}/balance", h.HandleGetBalance)

	return r
}
