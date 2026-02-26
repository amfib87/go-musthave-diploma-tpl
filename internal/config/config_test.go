package config

import (
	"flag"
	"os"
	"testing"
)

func TestInitialize(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "returns default config with empty strings",
			want: &Config{
				RunAddress:    "",
				DBURI:         "",
				AccSystemAddr: "",
			},
		},
		{
			name: "config fields are not nil",
			want: &Config{}, // неявная инициализация — все поля тоже пустые строки
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Initialize()

			// Проверка, что возвращён не nil
			if got == nil {
				t.Fatal("Initialize() returned nil, want non-nil *Config")
			}

			// Полевая проверка всех полей структуры
			if got.RunAddress != tt.want.RunAddress {
				t.Errorf("Initialize().RunAddress = %q, want %q",
					got.RunAddress, tt.want.RunAddress)
			}
			if got.DBURI != tt.want.DBURI {
				t.Errorf("Initialize().DBURI = %q, want %q",
					got.DBURI, tt.want.DBURI)
			}
			if got.AccSystemAddr != tt.want.AccSystemAddr {
				t.Errorf("Initialize().AccSystemAddr = %q, want %q",
					got.AccSystemAddr, tt.want.AccSystemAddr)
			}

			// Дополнительная проверка: убедимся, что все поля действительно пустые строки
			if got.RunAddress != "" {
				t.Errorf("Initialize().RunAddress should be empty string, got %q", got.RunAddress)
			}
			if got.DBURI != "" {
				t.Errorf("Initialize().DBURI should be empty string, got %q", got.DBURI)
			}
			if got.AccSystemAddr != "" {
				t.Errorf("Initialize().AccSystemAddr should be empty string, got %q", got.AccSystemAddr)
			}
		})
	}
}

func TestConfig_ParseFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		envVars  map[string]string
		expected *Config
	}{
		// ... тестовые случаи
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очистка флагов перед тестом
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Сохраняем исходные аргументы
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()
			os.Args = tt.args

			// Устанавливаем переменные окружения
			for key, value := range tt.envVars {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("failed to set env var %s: %v", key, err)
				}
			}
			defer func() {
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// Создаём конфигурацию и вызываем ParseFlag
			cf := Initialize()
			cf.ParseFlag()

			// Проверка результатов
			if cf.RunAddress != tt.expected.RunAddress {
				t.Errorf("RunAddress = %q, want %q", cf.RunAddress, tt.expected.RunAddress)
			}
			if cf.DBURI != tt.expected.DBURI {
				t.Errorf("DBURI = %q, want %q", cf.DBURI, tt.expected.DBURI)
			}
			if cf.AccSystemAddr != tt.expected.AccSystemAddr {
				t.Errorf("AccSystemAddr = %q, want %q", cf.AccSystemAddr, tt.expected.AccSystemAddr)
			}
		})
	}
}
