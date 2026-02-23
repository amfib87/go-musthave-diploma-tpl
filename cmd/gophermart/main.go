package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/config"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/handlers"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/router"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/storage"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {

	// Создаем логгер
	logger, err := logger.Initialize()
	if err != nil {
		log.Fatalf("failed init logger: %v", err)
		return fmt.Errorf("failed logger.Initialize: %v", err)
	}
	logger.Lg.Debug("logger was successfully created")

	// Парсим флаги
	cfg := config.Initialize()
	cfg.ParseFlag()
	logger.Lg.Debug("flags were parsed", zap.Any("cfg", cfg))
	// TODO 	// defer logger.Sync()

	// Инициализируем БД
	storage, err := storage.IniInitialize(cfg.DBURI)
	if err != nil {
		logger.Lg.Error("failed storage.IniInitialize:", zap.Error(err))
		return fmt.Errorf("failed storage.IniInitialize: %w", err)
	}
	// TODO DB.Close()

	// Инициализируем хэндлер
	handler := handlers.Initialize(logger, cfg, storage)

	// Инициализируем роутер
	router, err := router.Initialize(handler)

	logger.Lg.Info("running server", zap.String("cfg.RunAddress)", cfg.RunAddress))
	if err := http.ListenAndServe(cfg.RunAddress, router); err != nil {
		logger.Lg.Error("failed http.ListenAndServe: %v", zap.Error(err))
		return err
	}

	return nil
}
