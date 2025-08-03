package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"paymentSystem/internal/config"
	"paymentSystem/internal/handlers"
	logger2 "paymentSystem/internal/logger"
	"paymentSystem/internal/services"
	"paymentSystem/internal/storage/sqlite"
)

func main() {
	cfg := config.Load()
	fmt.Println(cfg.HTTPServer)

	logger := logger2.Init(cfg.Env)

	db, err := sql.Open("sqlite3", cfg.StoragePath)
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer db.Close()

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

	logger.Info("Server starting...")
	if err := srv.ListenAndServe(); err != nil {
		logger.Error("Server failed to start: ", err)
	}
}
