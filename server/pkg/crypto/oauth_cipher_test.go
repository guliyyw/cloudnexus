package crypto

import "testing"

func TestOAuthCipherEncryptDecrypt(t *testing.T) {
	cipher, err := NewOAuthCipher("Y2xvdWRuZXh1cy1kZXYtb2F1dGgta2V5LTMyYnl0ZXM=")
	if err != nil {
		t.Fatalf("NewOAuthCipher() error = %v", err)
	}

	encrypted, err := cipher.Encrypt("secret-token")
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encrypted == "secret-token" {
		t.Fatal("Encrypt() returned plaintext")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if decrypted != "secret-token" {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, "secret-token")
	}
}

func TestOAuthCipherDecryptRejectsInvalidPrefix(t *testing.T) {
	cipher, err := NewOAuthCipher("Y2xvdWRuZXh1cy1kZXYtb2F1dGgta2V5LTMyYnl0ZXM=")
	if err != nil {
		t.Fatalf("NewOAuthCipher() error = %v", err)
	}

	if _, err := cipher.Decrypt("plain-token"); err == nil {
		t.Fatal("Decrypt() expected error for invalid prefix")
	}
}
