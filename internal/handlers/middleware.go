// Пакет handlers содержит middleware для обработки запросов
//
// - Логирование всех запросов с метриками
// - Восстановление после паник (recovery)
package handlers

import (
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
	"runtime/debug"
	"time"
)

// LoggingMiddleware логирует информацию о каждом HTTP-запросе.
func (h *Handler) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww, ok := w.(middleware.WrapResponseWriter)
		if !ok {
			ww = middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		}

		start := time.Now()
		defer func() {
			h.logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration", time.Since(start),
			)
		}()

		next.ServeHTTP(ww, r)
	})
}

// RecoverMiddleware перехватывает паники во время обработки запросов.
func (h *Handler) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				stack := string(debug.Stack())
				h.logger.Error("panic", "error", err, "stack", stack)
				h.respondError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
