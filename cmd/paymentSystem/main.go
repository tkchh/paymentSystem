package main

import (
	"context"
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"paymentSystem/internal/config"
	"paymentSystem/internal/handlers"
	logger2 "paymentSystem/internal/logger"
	"paymentSystem/internal/services"
	"paymentSystem/internal/storage/sqlite"
	"syscall"
	"time"
)

func main() {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./config"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}

	logger := logger2.Init(cfg.Env)

	db, err := sql.Open("sqlite3", cfg.StoragePath)
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}

	storage := sqlite.NewStorage(db, logger)
	if err := storage.Init(); err != nil {
		log.Fatal("Storage init failed: ", err)
	}

	service := services.NewTransactionService(storage, logger)

	handler := handlers.NewHandler(service, logger)
	router := handlers.NewRouter(handler)

	srv := &http.Server{
		Addr:        cfg.Address,
		Handler:     router,
		ReadTimeout: cfg.Timeout,
		IdleTimeout: cfg.IdleTimeout,
	}

	go func() {
		logger.Info("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error: ", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	logger.Info("Shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown failed: ", err)
		srv.Close()
	}

	logger.Info("Closing database connection")
	if err := db.Close(); err != nil {
		logger.Error("Database close failed: ", err)

	}

	logger.Info("Server gracefully stopped")
}
