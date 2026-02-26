package storage

import (
	"bytes"
	"crypto/sha256"
	"testing"
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
