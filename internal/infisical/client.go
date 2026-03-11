package infisical

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Client holds the connection details for the Infisical API.
type Client struct {
	Host  string
	Token string
}

// NewClient creates a new Infisical API client.
func NewClient(host, token string) *Client {
	return &Client{
		Host:  host,
		Token: token,
	}
}

// GeneratePassword generates a 32-character random password that is URL-safe
// (no /, +, or = characters).
func GeneratePassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(b)
	// RawURLEncoding uses - and _ instead of + and /, and no padding.
	// Trim to 32 characters.
	if len(encoded) > 32 {
		encoded = encoded[:32]
	}
	return encoded, nil
}

// GenerateLaravelKey generates a Laravel-compatible application key in the
// format "base64:<32 bytes base64 encoded>".
func GenerateLaravelKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return "base64:" + base64.StdEncoding.EncodeToString(b), nil
}

type secretEntry struct {
	SecretKey   string `json:"secretKey"`
	SecretValue string `json:"secretValue"`
}

type secretPayload struct {
	ProjectSlug string        `json:"projectSlug"`
	Environment string        `json:"environment"`
	SecretPath  string        `json:"secretPath"`
	Secrets     []secretEntry `json:"secrets"`
}

// buildCreateSecretsPayload constructs the request payload for the batch
// secrets API.
func buildCreateSecretsPayload(secrets map[string]string, projectSlug, envSlug, secretsPath string) secretPayload {
	entries := make([]secretEntry, 0, len(secrets))
	for k, v := range secrets {
		entries = append(entries, secretEntry{
			SecretKey:   k,
			SecretValue: v,
		})
	}
	return secretPayload{
		ProjectSlug: projectSlug,
		Environment: envSlug,
		SecretPath:  secretsPath,
		Secrets:     entries,
	}
}

// CreateSecrets sends a batch of secrets to the Infisical API.
func (c *Client) CreateSecrets(secrets map[string]string, projectSlug, envSlug, secretsPath string) error {
	payload := buildCreateSecretsPayload(secrets, projectSlug, envSlug, secretsPath)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal secrets payload: %w", err)
	}

	url := strings.TrimRight(c.Host, "/") + "/api/v3/secrets/batch/raw"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("infisical API returned status %d", resp.StatusCode)
	}

	return nil
}
