package secrets

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// GenerateAppKey generates a Laravel-compatible APP_KEY (base64:...).
func GenerateAppKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("generating app key: %w", err)
	}
	return "base64:" + base64.StdEncoding.EncodeToString(key), nil
}

// GeneratePassword generates a random password of the given length using URL-safe base64.
func GeneratePassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}
