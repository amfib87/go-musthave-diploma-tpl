package logger

import (
	"fmt"

	"go.uber.org/zap"
)

type TLog struct {
	Lg *zap.Logger
}

// Инифиализация логера
func Initialize() (lg TLog, err error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return TLog{}, fmt.Errorf("failed NewDevelopment: %v", err)
	}

	return TLog{Lg: logger}, nil
}
