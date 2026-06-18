package service

import (
	"errors"
	"testing"

	"github.com/cloudnexus/server/pkg/model"
)

type stubTokenCipher struct {
	encryptFn func(string) (string, error)
	decryptFn func(string) (string, error)
}

func (s stubTokenCipher) Encrypt(plain string) (string, error) {
	if s.encryptFn != nil {
		return s.encryptFn(plain)
	}
	return plain, nil
}

func (s stubTokenCipher) Decrypt(encrypted string) (string, error) {
	if s.decryptFn != nil {
		return s.decryptFn(encrypted)
	}
	return encrypted, nil
}

func TestDecryptBindingSkipsLegacyPlaintext(t *testing.T) {
	svc := &OAuthService{
		cipher: stubTokenCipher{
			decryptFn: func(encrypted string) (string, error) {
				t.Fatalf("Decrypt should not be called for legacy plaintext binding")
				return "", nil
			},
		},
	}
	binding := &model.OAuthBinding{
		AccessToken:            "plain-access",
		RefreshToken:           "plain-refresh",
		TokenEncryptionVersion: 0,
	}

	if err := svc.decryptBinding(binding); err != nil {
		t.Fatalf("decryptBinding() error = %v", err)
	}
	if binding.AccessToken != "plain-access" || binding.RefreshToken != "plain-refresh" {
		t.Fatalf("legacy binding should remain unchanged, got access=%q refresh=%q", binding.AccessToken, binding.RefreshToken)
	}
}

func TestDecryptBindingDecryptsEncryptedTokens(t *testing.T) {
	svc := &OAuthService{
		cipher: stubTokenCipher{
			decryptFn: func(encrypted string) (string, error) {
			if encrypted == "enc-access" {
				return "plain-access", nil
			}
			if encrypted == "enc-refresh" {
				return "plain-refresh", nil
			}
			return "", errors.New("unexpected payload")
		},
		},
	}
	binding := &model.OAuthBinding{
		AccessToken:            "enc-access",
		RefreshToken:           "enc-refresh",
		TokenEncryptionVersion: 1,
	}

	if err := svc.decryptBinding(binding); err != nil {
		t.Fatalf("decryptBinding() error = %v", err)
	}
	if binding.AccessToken != "plain-access" || binding.RefreshToken != "plain-refresh" {
		t.Fatalf("unexpected decrypted values: access=%q refresh=%q", binding.AccessToken, binding.RefreshToken)
	}
}

func TestDecryptBindingReturnsErrorOnCipherFailure(t *testing.T) {
	svc := &OAuthService{
		cipher: stubTokenCipher{
			decryptFn: func(encrypted string) (string, error) {
			return "", errors.New("boom")
		},
		},
	}
	binding := &model.OAuthBinding{
		AccessToken:            "enc-access",
		RefreshToken:           "enc-refresh",
		TokenEncryptionVersion: 1,
	}

	if err := svc.decryptBinding(binding); err == nil {
		t.Fatal("decryptBinding() expected error when cipher fails")
	}
}
