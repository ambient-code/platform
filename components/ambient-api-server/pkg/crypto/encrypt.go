package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const (
	nonceSize  = 12
	minPayload = nonceSize + 16 // nonce + GCM tag minimum
)

type Keyring struct {
	keys          map[int]cipher.AEAD
	activeVersion int
}

func NewKeyring(keys map[string]string, activeVersion int) (*Keyring, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("keyring must contain at least one key")
	}

	aeadMap := make(map[int]cipher.AEAD, len(keys))
	seen := make(map[int]string, len(keys))
	for vStr, encoded := range keys {
		v, err := strconv.Atoi(vStr)
		if err != nil {
			return nil, fmt.Errorf("invalid key version %q: %w", vStr, err)
		}
		if prev, exists := seen[v]; exists {
			return nil, fmt.Errorf("duplicate key version %d: %q conflicts with %q", v, vStr, prev)
		}
		seen[v] = vStr
		raw, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("key version %d: invalid base64: %w", v, err)
		}
		if len(raw) != 32 {
			return nil, fmt.Errorf("key version %d: expected 32 bytes, got %d", v, len(raw))
		}
		block, err := aes.NewCipher(raw)
		if err != nil {
			return nil, fmt.Errorf("key version %d: %w", v, err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("key version %d: %w", v, err)
		}
		aeadMap[v] = aead
	}

	if _, ok := aeadMap[activeVersion]; !ok {
		return nil, fmt.Errorf("active version %d not found in keyring", activeVersion)
	}

	return &Keyring{keys: aeadMap, activeVersion: activeVersion}, nil
}

func ParseKeyringEnv(raw string, activeVersion int) (*Keyring, error) {
	var keys map[string]string
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, fmt.Errorf("parse keyring JSON: %w", err)
	}
	return NewKeyring(keys, activeVersion)
}

func (kr *Keyring) ActiveVersion() int {
	return kr.activeVersion
}

func (kr *Keyring) Versions() []int {
	versions := make([]int, 0, len(kr.keys))
	for v := range kr.keys {
		versions = append(versions, v)
	}
	return versions
}

func (kr *Keyring) Encrypt(plaintext, credentialID string) (string, error) {
	aead := kr.keys[kr.activeVersion]
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	aad := []byte(credentialID)
	ciphertext := aead.Seal(nonce, nonce, []byte(plaintext), aad)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return fmt.Sprintf("enc:v%d:%s", kr.activeVersion, encoded), nil
}

func (kr *Keyring) Decrypt(envelope, credentialID string) (string, error) {
	version, payload, err := parseEnvelope(envelope)
	if err != nil {
		return "", err
	}

	aead, ok := kr.keys[version]
	if !ok {
		return "", fmt.Errorf("key version %d not found in keyring", version)
	}

	if len(payload) < minPayload {
		return "", fmt.Errorf("ciphertext too short: %d bytes", len(payload))
	}

	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	aad := []byte(credentialID)

	plaintext, err := aead.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		return "", fmt.Errorf("decryption failed: %w", err)
	}

	return string(plaintext), nil
}

func parseEnvelope(s string) (int, []byte, error) {
	if !strings.HasPrefix(s, "enc:v") {
		return 0, nil, fmt.Errorf("not an encrypted envelope: missing enc:v prefix")
	}

	rest := s[len("enc:v"):]
	colonIdx := strings.IndexByte(rest, ':')
	if colonIdx < 1 {
		return 0, nil, fmt.Errorf("invalid envelope: missing version delimiter")
	}

	version, err := strconv.Atoi(rest[:colonIdx])
	if err != nil {
		return 0, nil, fmt.Errorf("invalid envelope: non-integer version %q", rest[:colonIdx])
	}

	b64 := rest[colonIdx+1:]
	payload, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return 0, nil, fmt.Errorf("invalid envelope: base64 decode failed: %w", err)
	}

	if len(payload) < minPayload {
		return 0, nil, fmt.Errorf("invalid envelope: payload too short (%d bytes, minimum %d)", len(payload), minPayload)
	}

	return version, payload, nil
}

func IsEncrypted(s string) bool {
	_, _, err := parseEnvelope(s)
	return err == nil
}

func TokenVersion(s string) (int, bool) {
	v, _, err := parseEnvelope(s)
	if err != nil {
		return 0, false
	}
	return v, true
}
