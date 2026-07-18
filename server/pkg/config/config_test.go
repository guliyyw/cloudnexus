package config

import "testing"

func TestValidateOAuthEncryptionKey(t *testing.T) {
	cfg := &AppConfig{}

	if err := cfg.ValidateOAuthEncryptionKey(); err == nil {
		t.Fatal("expected error for missing key")
	}

	cfg.OAuth.EncryptionKey = "invalid-base64"
	if err := cfg.ValidateOAuthEncryptionKey(); err == nil {
		t.Fatal("expected error for invalid base64 key")
	}

	cfg.OAuth.EncryptionKey = "Y2xvdWQ="
	if err := cfg.ValidateOAuthEncryptionKey(); err == nil {
		t.Fatal("expected error for invalid key length")
	}

	cfg.OAuth.EncryptionKey = "Y2xvdWRuZXh1cy1kZXYtb2F1dGgta2V5LTMyYnl0ZXM="
	if err := cfg.ValidateOAuthEncryptionKey(); err != nil {
		t.Fatalf("ValidateOAuthEncryptionKey() error = %v", err)
	}
}
