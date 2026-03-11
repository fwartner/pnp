package infisical

import (
	"encoding/json"
	"testing"
)

func TestGeneratePassword(t *testing.T) {
	pw, err := GeneratePassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pw) < 24 {
		t.Errorf("expected password length >= 24, got %d", len(pw))
	}

	// Verify no special base64 chars
	for _, c := range pw {
		if c == '/' || c == '+' || c == '=' {
			t.Errorf("password contains disallowed character: %c", c)
		}
	}

	// Verify uniqueness (two calls should not produce the same result)
	pw2, err := GeneratePassword()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw == pw2 {
		t.Error("two consecutive GeneratePassword calls returned the same value")
	}
}

func TestGenerateLaravelKey(t *testing.T) {
	key, err := GenerateLaravelKey()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(key) <= 10 {
		t.Errorf("expected key length > 10, got %d", len(key))
	}

	prefix := "base64:"
	if key[:len(prefix)] != prefix {
		t.Errorf("expected key to start with %q, got %q", prefix, key[:len(prefix)])
	}
}

func TestSecretPayload(t *testing.T) {
	secrets := map[string]string{
		"DB_HOST":     "localhost",
		"DB_PASSWORD": "secret123",
	}

	payload := buildCreateSecretsPayload(secrets, "my-project", "production", "/")

	if payload.ProjectSlug != "my-project" {
		t.Errorf("expected projectSlug 'my-project', got %q", payload.ProjectSlug)
	}
	if payload.Environment != "production" {
		t.Errorf("expected environment 'production', got %q", payload.Environment)
	}
	if payload.SecretPath != "/" {
		t.Errorf("expected secretPath '/', got %q", payload.SecretPath)
	}
	if len(payload.Secrets) != 2 {
		t.Fatalf("expected 2 secrets, got %d", len(payload.Secrets))
	}

	// Build a lookup for verification
	found := make(map[string]string)
	for _, s := range payload.Secrets {
		found[s.SecretKey] = s.SecretValue
	}

	if found["DB_HOST"] != "localhost" {
		t.Errorf("expected DB_HOST=localhost, got %q", found["DB_HOST"])
	}
	if found["DB_PASSWORD"] != "secret123" {
		t.Errorf("expected DB_PASSWORD=secret123, got %q", found["DB_PASSWORD"])
	}

	// Verify JSON serialization uses correct field names
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}

	if _, ok := raw["projectSlug"]; !ok {
		t.Error("expected JSON key 'projectSlug'")
	}
	if _, ok := raw["environment"]; !ok {
		t.Error("expected JSON key 'environment'")
	}
	if _, ok := raw["secretPath"]; !ok {
		t.Error("expected JSON key 'secretPath'")
	}
	if _, ok := raw["secrets"]; !ok {
		t.Error("expected JSON key 'secrets'")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("https://infisical.example.com", "my-token")
	if c.Host != "https://infisical.example.com" {
		t.Errorf("unexpected host: %q", c.Host)
	}
	if c.Token != "my-token" {
		t.Errorf("unexpected token: %q", c.Token)
	}
}
