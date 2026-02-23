package gzip

import (
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestNewCompressReader(t *testing.T) {
	tests := []struct {
		name    string
		r       io.ReadCloser
		wantErr bool
	}{
		{
			name: "successful creation with valid gzip data",
			r: func() io.ReadCloser {
				var buf bytes.Buffer
				zw := gzip.NewWriter(&buf)
				_, _ = zw.Write([]byte("test data"))
				_ = zw.Close()
				return io.NopCloser(bytes.NewReader(buf.Bytes()))
			}(),
			wantErr: false,
		},
		{
			name:    "error when creating reader from invalid gzip data",
			r:       io.NopCloser(bytes.NewReader([]byte("not gzip data"))),
			wantErr: true,
		},
		{
			name:    "nil reader should return error",
			r:       nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := NewCompressReader(tt.r)

			if tt.wantErr {
				if gotErr == nil {
					t.Fatal("NewCompressReader() succeeded unexpectedly, expected error")
				}
				// Для nil-входа достаточно проверить факт ошибки
				return
			}

			if gotErr != nil {
				t.Fatalf("NewCompressReader() failed: %v", gotErr)
			}

			// Базовая проверка на не-nil результат
			if got == nil {
				t.Fatal("NewCompressReader() returned nil, want non-nil *compressReader")
			}

			// Дополнительная проверка: убедимся, что zr инициализирован
			if got.zr == nil {
				t.Error("compressReader.zr is nil, want initialized gzip.Reader")
			}

			// Проверка чтения данных (только для успешного случая)
			data := make([]byte, 100)
			n, err := got.Read(data)
			if err != nil && err != io.EOF {
				t.Errorf("Read() returned unexpected error: %v", err)
			}

			if n > 0 {
				readData := string(data[:n])
				if readData != "test data" && readData != "" {
					t.Errorf("Read() = %q, want 'test data' or empty (EOF)", readData)
				}
			}

			// Закрытие ресурса
			if closeErr := got.Close(); closeErr != nil {
				t.Errorf("Close() returned error: %v", closeErr)
			}
		})
	}
}
