package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/amfib87/go-musthave-diploma-tpl/internal/logger"
	"go.uber.org/zap"
)

func TestGetHash(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{
			name:  "empty_string",
			value: "",
		},
		{
			name:  "simple_string",
			value: "hello",
		},
		{
			name:  "long_string",
			value: "This is a longer test string for hash function",
		},
		{
			name:  "string_with_special_chars",
			value: "test@123#$%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHash(tt.value)
			want := expectedHash(tt.value)

			if !bytes.Equal(got, want) {
				t.Errorf("GetHash(%q) = %v, want %v", tt.value, got, want)
			}
		})
	}
}

// expectedHash — вспомогательная функция для генерации эталонного хеша
func expectedHash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:]
}

func TestLuhnCheck(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "valid_16_digit_card",
			value: "4532015112830366",
			want:  true,
		},
		{
			name:  "invalid_16_digit_card",
			value: "4532015112830367",
			want:  false,
		},
		{
			name:  "valid_short_number",
			value: "79927398713",
			want:  true,
		},
		{
			name:  "invalid_short_number",
			value: "79927398714",
			want:  false,
		},
		{
			name:  "single_digit_valid",
			value: "0",
			want:  true,
		},
		{
			name:  "single_digit_invalid",
			value: "1",
			want:  false,
		},
		{
			name:  "two_digits_valid",
			value: "45",  //
			want:  false, //
		},
		{
			name:  "all_zeros",
			value: "0000",
			want:  true,
		},
		{
			name:  "empty_string",
			value: "",
			want:  true,
		},
		{
			name:  "non_digit_characters",
			value: "123a",
			want:  false,
		},
		{
			name:  "leading_zeros",
			value: "004532015112830366",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LuhnCheck(tt.value)
			if got != tt.want {
				t.Errorf("LuhnCheck(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestGetAccrualDataOrder(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		order       string
		handler     http.HandlerFunc
		want        Accrual
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful response with accrual data",
			baseURL: "http://localhost:8080",
			order:   "12345",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(Accrual{
					Order:   "12345",
					Status:  "processed",
					Accrual: 150.5,
				})
			},
			want: Accrual{
				Order:   "12345",
				Status:  "processed",
				Accrual: 150.5,
			},
			wantErr: false,
		},
		{
			name:    "order not found (NoContent)",
			baseURL: "http://localhost:8080",
			order:   "99999",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			want:        Accrual{},
			wantErr:     true,
			errContains: "order didn't find",
		},
		{
			name:    "too many requests with Retry-After",
			baseURL: "http://localhost:8080",
			order:   "67890",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
			},
			want:        Accrual{},
			wantErr:     true,
			errContains: "Must be repeated in 60 seconds",
		},
		{
			name:    "internal server error",
			baseURL: "http://localhost:8080",
			order:   "54321",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			want:        Accrual{},
			wantErr:     true,
			errContains: "internal server error",
		},
		{
			name:    "unknown HTTP status",
			baseURL: "http://localhost:8080",
			order:   "11111",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadGateway)
			},
			want:        Accrual{},
			wantErr:     true,
			errContains: "unknown HTTP status 502",
		},
		{
			name:    "empty Retry-After header",
			baseURL: "http://localhost:8080",
			order:   "33333",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusTooManyRequests)
			},
			want:        Accrual{},
			wantErr:     true,
			errContains: "Must be repeated in unknown seconds",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			log := logger.TLog{
				Lg: zap.NewNop(),
			}

			got, gotErr := GetAccrualDataOrder(server.URL, tt.order, log)

			if tt.wantErr {
				if gotErr == nil {
					t.Fatal("expected error but got none")
				}
				if tt.errContains != "" && !strings.Contains(gotErr.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, gotErr.Error())
				}
				return
			}

			if gotErr != nil {
				t.Fatalf("unexpected error: %v", gotErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
