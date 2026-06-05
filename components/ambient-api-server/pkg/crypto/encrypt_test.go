package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
)

func testKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func testKeyring(t *testing.T) *Keyring {
	t.Helper()
	key := base64.StdEncoding.EncodeToString(testKey())
	kr, err := NewKeyring(map[string]string{"1": key}, 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	return kr
}

func TestNewKeyring_Valid(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(testKey())
	kr, err := NewKeyring(map[string]string{"1": key}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kr.ActiveVersion() != 1 {
		t.Fatalf("expected active version 1, got %d", kr.ActiveVersion())
	}
}

func TestNewKeyring_InvalidKeyLength(t *testing.T) {
	short := base64.StdEncoding.EncodeToString([]byte("tooshort"))
	_, err := NewKeyring(map[string]string{"1": short}, 1)
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestNewKeyring_InvalidBase64(t *testing.T) {
	_, err := NewKeyring(map[string]string{"1": "not-valid-base64!!!"}, 1)
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestNewKeyring_MissingActiveVersion(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(testKey())
	_, err := NewKeyring(map[string]string{"1": key}, 2)
	if err == nil {
		t.Fatal("expected error for missing active version")
	}
}

func TestNewKeyring_EmptyKeys(t *testing.T) {
	_, err := NewKeyring(map[string]string{}, 1)
	if err == nil {
		t.Fatal("expected error for empty keyring")
	}
}

func TestParseKeyringEnv(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(testKey())
	raw := `{"1":"` + key + `"}`
	kr, err := ParseKeyringEnv(raw, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kr.ActiveVersion() != 1 {
		t.Fatalf("expected active version 1, got %d", kr.ActiveVersion())
	}
}

func TestParseKeyringEnv_InvalidJSON(t *testing.T) {
	_, err := ParseKeyringEnv("{invalid", 1)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEncryptDecrypt_Roundtrip(t *testing.T) {
	kr := testKeyring(t)
	plaintext := "ghp_abc123_my_secret_token"
	credentialID := "cred-001"

	ciphertext, err := kr.Encrypt(plaintext, credentialID)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if !strings.HasPrefix(ciphertext, "enc:v1:") {
		t.Fatalf("expected enc:v1: prefix, got %q", ciphertext[:20])
	}

	decrypted, err := kr.Decrypt(ciphertext, credentialID)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("roundtrip failed: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	kr := testKeyring(t)
	plaintext := "same_value"
	credentialID := "cred-001"

	ct1, _ := kr.Encrypt(plaintext, credentialID)
	ct2, _ := kr.Encrypt(plaintext, credentialID)

	if ct1 == ct2 {
		t.Fatal("two encryptions of the same plaintext must produce different ciphertexts (random nonce)")
	}
}

func TestDecrypt_WrongCredentialID(t *testing.T) {
	kr := testKeyring(t)
	ciphertext, _ := kr.Encrypt("secret", "cred-001")

	_, err := kr.Decrypt(ciphertext, "cred-002")
	if err == nil {
		t.Fatal("expected error when decrypting with wrong credential ID (AAD mismatch)")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	kr := testKeyring(t)
	ciphertext, _ := kr.Encrypt("secret", "cred-001")

	parts := strings.SplitN(ciphertext, ":", 3)
	decoded, _ := base64.StdEncoding.DecodeString(parts[2])
	decoded[len(decoded)-1] ^= 0xFF
	parts[2] = base64.StdEncoding.EncodeToString(decoded)
	tampered := strings.Join(parts, ":")

	_, err := kr.Decrypt(tampered, "cred-001")
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_InvalidEnvelopePrefix(t *testing.T) {
	kr := testKeyring(t)
	_, err := kr.Decrypt("notenc:v1:data", "cred-001")
	if err == nil {
		t.Fatal("expected error for non-enc prefix")
	}
}

func TestDecrypt_InvalidVersionFormat(t *testing.T) {
	kr := testKeyring(t)
	_, err := kr.Decrypt("enc:vX:data", "cred-001")
	if err == nil {
		t.Fatal("expected error for non-integer version")
	}
}

func TestDecrypt_MissingKeyVersion(t *testing.T) {
	kr := testKeyring(t)
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 100)
	}

	kr2, _ := NewKeyring(map[string]string{
		"2": base64.StdEncoding.EncodeToString(key2),
	}, 2)

	ciphertext, _ := kr2.Encrypt("secret", "cred-001")

	_, err := kr.Decrypt(ciphertext, "cred-001")
	if err == nil {
		t.Fatal("expected error for missing key version")
	}
}

func TestDecrypt_InvalidBase64Payload(t *testing.T) {
	kr := testKeyring(t)
	_, err := kr.Decrypt("enc:v1:not-valid-base64!!!", "cred-001")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecrypt_PayloadTooShort(t *testing.T) {
	kr := testKeyring(t)
	short := base64.StdEncoding.EncodeToString([]byte("tiny"))
	_, err := kr.Decrypt("enc:v1:"+short, "cred-001")
	if err == nil {
		t.Fatal("expected error for payload shorter than nonce + tag")
	}
}

func TestIsEncrypted(t *testing.T) {
	kr := testKeyring(t)
	ct, _ := kr.Encrypt("secret", "cred-001")

	if !IsEncrypted(ct) {
		t.Fatal("expected IsEncrypted=true for valid ciphertext")
	}
	if IsEncrypted("ghp_abc123") {
		t.Fatal("expected IsEncrypted=false for plaintext")
	}
	if IsEncrypted("") {
		t.Fatal("expected IsEncrypted=false for empty string")
	}
	if IsEncrypted("enc:vX:bad") {
		t.Fatal("expected IsEncrypted=false for non-integer version")
	}
	if IsEncrypted("enc:v1") {
		t.Fatal("expected IsEncrypted=false for missing payload")
	}
}

func TestMultiVersionKeyring(t *testing.T) {
	key1 := testKey()
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 50)
	}

	kr, err := NewKeyring(map[string]string{
		"1": base64.StdEncoding.EncodeToString(key1),
		"2": base64.StdEncoding.EncodeToString(key2),
	}, 2)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}

	ct, err := kr.Encrypt("secret", "cred-001")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(ct, "enc:v2:") {
		t.Fatalf("expected v2 prefix, got %q", ct[:10])
	}

	// Can decrypt v2
	decrypted, err := kr.Decrypt(ct, "cred-001")
	if err != nil {
		t.Fatalf("Decrypt v2: %v", err)
	}
	if decrypted != "secret" {
		t.Fatalf("v2 roundtrip failed")
	}

	// Create a v1 ciphertext using a v1-only keyring and verify the multi-keyring can decrypt it
	kr1, _ := NewKeyring(map[string]string{"1": base64.StdEncoding.EncodeToString(key1)}, 1)
	ctV1, _ := kr1.Encrypt("old_secret", "cred-002")

	decryptedV1, err := kr.Decrypt(ctV1, "cred-002")
	if err != nil {
		t.Fatalf("Decrypt v1 with multi-keyring: %v", err)
	}
	if decryptedV1 != "old_secret" {
		t.Fatalf("v1 decrypt failed")
	}
}

func TestEncrypt_EmptyPlaintext(t *testing.T) {
	kr := testKeyring(t)
	ct, err := kr.Encrypt("", "cred-001")
	if err != nil {
		t.Fatalf("Encrypt empty: %v", err)
	}
	dec, err := kr.Decrypt(ct, "cred-001")
	if err != nil {
		t.Fatalf("Decrypt empty: %v", err)
	}
	if dec != "" {
		t.Fatalf("expected empty string, got %q", dec)
	}
}

func TestVersions(t *testing.T) {
	key1 := testKey()
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 50)
	}

	kr, err := NewKeyring(map[string]string{
		"1": base64.StdEncoding.EncodeToString(key1),
		"3": base64.StdEncoding.EncodeToString(key2),
	}, 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}

	versions := kr.Versions()
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	has1, has3 := false, false
	for _, v := range versions {
		if v == 1 {
			has1 = true
		}
		if v == 3 {
			has3 = true
		}
	}
	if !has1 || !has3 {
		t.Fatalf("expected versions [1, 3], got %v", versions)
	}
}

func TestTokenVersion(t *testing.T) {
	kr := testKeyring(t)
	ct, err := kr.Encrypt("secret", "cred-001")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantVer int
		wantOK  bool
	}{
		{"valid ciphertext", ct, 1, true},
		{"plaintext PAT", "ghp_abc123", 0, false},
		{"empty string", "", 0, false},
		{"partial prefix", "enc:v1", 0, false},
		{"non-integer version", "enc:vX:data", 0, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, ok := TokenVersion(tc.input)
			if ok != tc.wantOK {
				t.Fatalf("TokenVersion(%q): ok=%v, want %v", tc.input, ok, tc.wantOK)
			}
			if v != tc.wantVer {
				t.Fatalf("TokenVersion(%q): version=%d, want %d", tc.input, v, tc.wantVer)
			}
		})
	}
}

func TestNewKeyring_DuplicateVersions(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(testKey())
	key2 := make([]byte, 32)
	for i := range key2 {
		key2[i] = byte(i + 99)
	}
	_, err := NewKeyring(map[string]string{
		"1":  key,
		"01": base64.StdEncoding.EncodeToString(key2),
	}, 1)
	if err == nil {
		t.Fatal("expected error for duplicate version (1 and 01)")
	}
	if !strings.Contains(err.Error(), "duplicate key version") {
		t.Fatalf("expected duplicate error, got: %v", err)
	}
}

func TestEncrypt_LargePayload(t *testing.T) {
	kr := testKeyring(t)
	large := strings.Repeat("a]kubeconfig-content-here[", 1000)
	ct, err := kr.Encrypt(large, "cred-001")
	if err != nil {
		t.Fatalf("Encrypt large: %v", err)
	}
	dec, err := kr.Decrypt(ct, "cred-001")
	if err != nil {
		t.Fatalf("Decrypt large: %v", err)
	}
	if dec != large {
		t.Fatal("large payload roundtrip failed")
	}
}
