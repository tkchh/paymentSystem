// Пакет logger отвечает за инициализацию и настройку системы логирования.
package logger

import (
	"log/slog"
	"os"
)

// Init инициализирует и возвращает логгер с настройками для указанного окружения.
//
// В зависимости от окружения настраивает:
// - Формат вывода (текстовый для разработки, JSON для production)
// - Уровень логирования (Debug для разработки, Info для production)
func Init(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case "development":
		logger = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case "production":
		logger = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
		//JSON для обработки в продакшене
	}

	return logger
}
