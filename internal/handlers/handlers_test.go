package handlers

import (
	"context"
	"reflect"
	"testing"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/config"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"github.com/amfib87/go-musthave-diploma-tpl/internal/storage"
	"go.uber.org/zap"
)

type mockStorage struct {
	insertNewRecordUserFunc     func(ctx context.Context, data storage.LineUser) (string, error)
	insertNewRecordOrderFunc    func(ctx context.Context, data storage.LineOrder) (string, error)
	getDataUserFunc             func(ctx context.Context, login string) (storage.LineUser, error)
	getDataOrderFunc            func(ctx context.Context, order string) (storage.LineOrder, error)
	getDataAllOrdersFunc        func(ctx context.Context, login string) ([]storage.LineOrder, error)
	insertNewRecordWithdrawFunc func(ctx context.Context, login, order string, sum float64) error
	getAllUserWithdrawFunc      func(ctx context.Context, login string) ([]storage.LineWithdraw, error)
	closeFunc                   func() error
}

func (m *mockStorage) InsertNewRecordUser(ctx context.Context, data storage.LineUser) (string, error) {
	if m.insertNewRecordUserFunc != nil {
		return m.insertNewRecordUserFunc(ctx, data)
	}
	return "", nil
}

func (m *mockStorage) InsertNewRecordOrder(ctx context.Context, data storage.LineOrder) (string, error) {
	if m.insertNewRecordOrderFunc != nil {
		return m.insertNewRecordOrderFunc(ctx, data)
	}
	return "", nil
}

func (m *mockStorage) GetDataUser(ctx context.Context, login string) (storage.LineUser, error) {
	if m.getDataUserFunc != nil {
		return m.getDataUserFunc(ctx, login)
	}
	return storage.LineUser{}, nil
}

func (m *mockStorage) GetDataOrder(ctx context.Context, order string) (storage.LineOrder, error) {
	if m.getDataOrderFunc != nil {
		return m.getDataOrderFunc(ctx, order)
	}
	return storage.LineOrder{}, nil
}

func (m *mockStorage) GetDataAllOrders(ctx context.Context, login string) ([]storage.LineOrder, error) {
	if m.getDataAllOrdersFunc != nil {
		return m.getDataAllOrdersFunc(ctx, login)
	}
	return nil, nil
}

func (m *mockStorage) InsertNewRecordWithdraw(ctx context.Context, login, order string, sum float64) error {
	if m.insertNewRecordWithdrawFunc != nil {
		m.insertNewRecordWithdrawFunc(ctx, login, order, sum)
	}
	return nil
}

func (m *mockStorage) GetAllUserWithdraw(ctx context.Context, login string) ([]storage.LineWithdraw, error) {
	if m.getAllUserWithdrawFunc != nil {
		return m.getAllUserWithdrawFunc(ctx, login)
	}
	return nil, nil
}

func (m *mockStorage) Close() error {
	if m.getAllUserWithdrawFunc != nil {
		return m.Close()
	}
	return nil
}

func TestInitialize(t *testing.T) {
	tests := []struct {
		name string
		lg   logger.TLog
		cfg  *config.Config
		st   storage.Storage
		want *Handler
	}{
		{
			name: "normal initialization with full config",
			lg:   logger.TLog{Lg: zap.NewNop()},
			cfg: &config.Config{
				RunAddress:    "localhost:8080",
				DBURI:         "postgres://user:pass@localhost/db",
				AccSystemAddr: "http://acc-system:9000",
			},
			st: &mockStorage{},
			want: &Handler{
				Log: logger.TLog{Lg: zap.NewNop()},
				Cfg: &config.Config{
					RunAddress:    "localhost:8080",
					DBURI:         "postgres://user:pass@localhost/db",
					AccSystemAddr: "http://acc-system:9000",
				},
			},
		},

		{
			name: "minimal config values",
			lg:   logger.TLog{Lg: zap.NewNop()},
			cfg: &config.Config{
				RunAddress:    "",
				DBURI:         "",
				AccSystemAddr: "",
			},
			st: &mockStorage{},
			want: &Handler{
				Log: logger.TLog{Lg: zap.NewNop()},
				Cfg: &config.Config{
					RunAddress:    "",
					DBURI:         "",
					AccSystemAddr: "",
				},
			},
		},
		{
			name: "nil config",
			lg:   logger.TLog{Lg: zap.NewNop()},
			cfg:  nil,
			st:   &mockStorage{},
			want: &Handler{
				Log: logger.TLog{Lg: zap.NewNop()},
				Cfg: nil,
			},
		},
		{
			name: "empty logger",
			lg:   logger.TLog{},
			cfg:  &config.Config{RunAddress: "localhost:8080"},
			st:   &mockStorage{},
			want: &Handler{
				Log: logger.TLog{},
				Cfg: &config.Config{RunAddress: "localhost:8080"},
			},
		},

		{
			name: "all nil values",
			lg:   logger.TLog{},
			cfg:  nil,
			st:   &mockStorage{},
			want: &Handler{
				Log: logger.TLog{},
				Cfg: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Initialize(tt.lg, tt.cfg, tt.st)

			// Проверка поля Log — используем reflect.DeepEqual для структур
			if !reflect.DeepEqual(got.Log, tt.want.Log) {
				t.Errorf("Initialize().Log = %+v, want %+v", got.Log, tt.want.Log)
			}

			// Проверка поля Cfg: сначала статус nil, затем содержимое
			if got.Cfg == nil && tt.want.Cfg != nil {
				t.Errorf("Initialize().Cfg is nil, but expected non-nil value with: %+v", tt.want.Cfg)
			} else if got.Cfg != nil && tt.want.Cfg == nil {
				t.Errorf("Initialize().Cfg is non-nil, but expected nil")
			} else if got.Cfg != nil && tt.want.Cfg != nil {
				// Сравниваем поля только если оба Cfg не nil
				if got.Cfg.RunAddress != tt.want.Cfg.RunAddress {
					t.Errorf("Initialize().Cfg.RunAddress = %q, want %q", got.Cfg.RunAddress, tt.want.Cfg.RunAddress)
				}
				if got.Cfg.DBURI != tt.want.Cfg.DBURI {
					t.Errorf("Initialize().Cfg.DBURI = %q, want %q", got.Cfg.DBURI, tt.want.Cfg.DBURI)
				}
				if got.Cfg.AccSystemAddr != tt.want.Cfg.AccSystemAddr {
					t.Errorf("Initialize().Cfg.AccSystemAddr = %q, want %q", got.Cfg.AccSystemAddr, tt.want.Cfg.AccSystemAddr)
				}
			}

			// Проверка поля Stor — должно быть не nil
			if got.Stor == nil {
				t.Error("Initialize().Stor is nil, but expected non-nil value")
			}
		})
	}

}
