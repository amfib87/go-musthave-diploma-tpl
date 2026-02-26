package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "successful initialization with development logger",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := Initialize()

			if tt.wantErr {
				if gotErr == nil {
					t.Fatal("Initialize() succeeded unexpectedly, expected error")
				}
				return
			}

			if gotErr != nil {
				t.Fatalf("Initialize() failed: %v", gotErr)
			}

			// Проверка, что logger не nil
			if got.Lg == nil {
				t.Fatal("Initialize() returned nil logger, want non-nil *zap.Logger")
			}

			// Базовая проверка конфигурации логгера (опционально)
			// Можно проверить, что это действительно development-конфигурация
			atomicLevel := got.Lg.Level()
			if atomicLevel.Enabled(zap.DebugLevel) {
				// В development mode обычно включён debug level
			} else {
				t.Errorf("logger level is not debug-enabled, want debug level for development")
			}

		})
	}
}
