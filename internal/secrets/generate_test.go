package secrets

import (
	"strings"
	"testing"
)

func TestGenerateAppKey(t *testing.T) {
	key, err := GenerateAppKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(key, "base64:") {
		t.Errorf("expected key to start with 'base64:', got %s", key)
	}
	// base64 of 32 bytes = 44 chars + "base64:" prefix = 51
	if len(key) != 51 {
		t.Errorf("expected key length 51, got %d", len(key))
	}

	// Ensure uniqueness
	key2, _ := GenerateAppKey()
	if key == key2 {
		t.Error("two generated keys should not be equal")
	}
}

func TestGeneratePassword(t *testing.T) {
	pw, err := GeneratePassword(32)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) != 32 {
		t.Errorf("expected password length 32, got %d", len(pw))
	}

	pw2, _ := GeneratePassword(32)
	if pw == pw2 {
		t.Error("two generated passwords should not be equal")
	}
}

func TestGeneratePassword_Length(t *testing.T) {
	for _, length := range []int{8, 16, 24, 48} {
		pw, err := GeneratePassword(length)
		if err != nil {
			t.Fatalf("unexpected error for length %d: %v", length, err)
		}
		if len(pw) != length {
			t.Errorf("expected length %d, got %d", length, len(pw))
		}
	}
}
