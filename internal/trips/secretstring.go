package trips

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"strings"
)

// secretEncPrefix is prepended to AES-GCM–encrypted app settings values (base64 payload).
const secretEncPrefix = "enc1:"

// settingsEncryptionKey returns 32 bytes from REMI_SETTINGS_ENCRYPTION_KEY, or nil if unset.
// Key may be: 64-char hex, or any string (SHA-256 is used) for convenience.
func settingsEncryptionKey() []byte {
	raw := strings.TrimSpace(os.Getenv("REMI_SETTINGS_ENCRYPTION_KEY"))
	if raw == "" {
		return nil
	}
	if len(raw) == 64 {
		if b, err := hex.DecodeString(raw); err == nil && len(b) == 32 {
			return b
		}
	}
	if len(raw) == 32 {
		return []byte(raw)
	}
	s := sha256.Sum256([]byte(raw))
	return s[:]
}

// EncryptAppSettingSecret returns ciphertext when REMI_SETTINGS_ENCRYPTION_KEY is set, else plain.
func EncryptAppSettingSecret(plain string) string {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	key := settingsEncryptionKey()
	if key == nil {
		return plain
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return plain
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return plain
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return plain
	}
	ct := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return secretEncPrefix + base64.RawStdEncoding.EncodeToString(ct)
}

// DecryptAppSettingSecret reverses EncryptAppSettingSecret; unknown / legacy values pass through.
func DecryptAppSettingSecret(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" || !strings.HasPrefix(stored, secretEncPrefix) {
		return stored
	}
	key := settingsEncryptionKey()
	if key == nil {
		// No key: cannot decrypt; leave opaque (treat as empty for API use).
		return ""
	}
	b, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(stored, secretEncPrefix))
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	if len(b) < gcm.NonceSize() {
		return ""
	}
	nonce, ciph := b[:gcm.NonceSize()], b[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciph, nil)
	if err != nil {
		return ""
	}
	return string(plain)
}
