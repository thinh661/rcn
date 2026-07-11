package services

import (
	"fmt"

	"github.com/rcn/rcn/backend/internal/config"
	"github.com/rcn/rcn/backend/internal/database"
)

// SecretVault provides a high-level encrypted key-value store for app secrets
// backed by the app_secrets table. All values are encrypted at rest using
// AES-256-GCM (via Encrypt/Decrypt from crypto.go) with a key derived from
// the JWT secret. The vault tracks a rotation_version that increments each
// time a value is re-encrypted, providing an audit signal for key rotation.
type SecretVault struct {
	encryptionKey []byte // passed through deriveKey internally for AES-256
}

// NewSecretVault creates a vault whose encryption key is derived from the
// JWT secret key (the same key used by the MinIO IAM encryption layer).
func NewSecretVault(cfg *config.Config) *SecretVault {
	return &SecretVault{encryptionKey: []byte(cfg.JWTSecretKey)}
}

// SetSecret encrypts the value and upserts it into app_secrets. If the key
// already exists, rotation_version is incremented. Returns an error if the
// encryption or DB write fails.
func (v *SecretVault) SetSecret(key, value string) error {
	if key == "" {
		return fmt.Errorf("secret key cannot be empty")
	}

	enc, err := Encrypt(value, v.encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	_, err = database.GetDB().Exec(`
		INSERT INTO app_secrets (key, value, rotation_version)
		VALUES ($1, $2, 1)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			rotation_version = app_secrets.rotation_version + 1
	`, key, enc)
	if err != nil {
		return fmt.Errorf("db upsert secret: %w", err)
	}
	return nil
}

// GetSecret retrieves and decrypts a secret value. Returns an error if the
// key does not exist or decryption fails.
func (v *SecretVault) GetSecret(key string) (string, error) {
	var enc string
	err := database.GetDB().QueryRow(
		`SELECT value FROM app_secrets WHERE key = $1`, key,
	).Scan(&enc)
	if err != nil {
		return "", fmt.Errorf("secret %q not found: %w", key, err)
	}

	plaintext, err := Decrypt(enc, v.encryptionKey)
	if err != nil {
		return "", fmt.Errorf("decrypt secret %q: %w", key, err)
	}
	return plaintext, nil
}

// DeleteSecret removes a secret from the store. Returns an error if the key
// does not exist.
func (v *SecretVault) DeleteSecret(key string) error {
	res, err := database.GetDB().Exec(`DELETE FROM app_secrets WHERE key = $1`, key)
	if err != nil {
		return fmt.Errorf("db delete secret: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("secret %q not found", key)
	}
	return nil
}

// ListSecrets returns all secret key names in alphabetical order. Values are
// never returned — this is a key-names-only operation.
func (v *SecretVault) ListSecrets() ([]string, error) {
	rows, err := database.GetDB().Query(`SELECT key FROM app_secrets ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("db list secrets: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("scan secret key: %w", err)
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RotateAllSecrets re-encrypts every secret with the current encryption key
// and increments each rotation_version. The encryption key itself has not
// changed (that would require changing the JWT secret), but this provides the
// cryptographic re-wrap step that a full key-rotation procedure would call.
// Returns the number of secrets rotated and any error encountered.
func (v *SecretVault) RotateAllSecrets() (int, error) {
	keys, err := v.ListSecrets()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, k := range keys {
		plaintext, err := v.GetSecret(k)
		if err != nil {
			return count, fmt.Errorf("rotate %q: read: %w", k, err)
		}
		if err := v.SetSecret(k, plaintext); err != nil {
			return count, fmt.Errorf("rotate %q: write: %w", k, err)
		}
		count++
	}
	return count, nil
}
