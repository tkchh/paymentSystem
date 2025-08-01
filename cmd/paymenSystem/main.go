package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"paymentSystem/internal/config"
	logger2 "paymentSystem/internal/logger"
	"paymentSystem/internal/storage/sqlite"
)

func main() {
	cfg := config.Load()
	fmt.Println(cfg.HTTPServer)

	logger := logger2.Init(cfg.Env)

	logger.Warn(fmt.Sprintf("Config: %v", cfg))

	db, err := sql.Open("sqlite3", cfg.StoragePath)
	if err != nil {
		log.Fatal("Database connection failed: ", err)
	}
	defer db.Close()

	storage := sqlite.NewStorage(db, logger)
	if err := storage.Init(); err != nil {
		log.Fatal("Storage init failed: ", err)
	}

	err = storage.Transfer("eb16dae1-9c4d-40fa-bd14-d6bc11207a87", "0d439904-bb31-4d56-bbe0-5719f319248f", 0.002)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Hello World")
}
