package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const oauthCipherPrefix = "enc:v1:"

type OAuthCipher struct {
	key []byte
}

func NewOAuthCipher(encodedKey string) (*OAuthCipher, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode oauth encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("oauth encryption key must decode to 32 bytes")
	}
	return &OAuthCipher{key: key}, nil
}

func (c *OAuthCipher) Encrypt(plain string) (string, error) {
	if plain == "" {
		return "", nil
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	payload := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return oauthCipherPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func (c *OAuthCipher) Decrypt(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	if !strings.HasPrefix(encrypted, oauthCipherPrefix) {
		return "", fmt.Errorf("invalid oauth token prefix")
	}
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(encrypted, oauthCipherPrefix))
	if err != nil {
		return "", fmt.Errorf("decode oauth token payload: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", fmt.Errorf("create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("create gcm: %w", err)
	}
	if len(payload) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid oauth token payload")
	}
	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt oauth token: %w", err)
	}
	return string(plain), nil
}
