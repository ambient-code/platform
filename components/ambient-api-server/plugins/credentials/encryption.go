package credentials

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/crypto"
	"gorm.io/gorm"
)

const (
	envKeyring        = "CREDENTIAL_ENCRYPTION_KEYRING"
	envKeyVersion     = "CREDENTIAL_ENCRYPTION_KEY_VERSION"
	envAllowPlaintext = "CREDENTIAL_ENCRYPTION_ALLOW_PLAINTEXT"
)

func LoadKeyring() *crypto.Keyring {
	raw := os.Getenv(envKeyring)
	if raw == "" {
		if !IsPlaintextAllowed() {
			fmt.Fprintf(os.Stderr, "FATAL: credential encryption disabled — set %s or set %s=true to override\n", envKeyring, envAllowPlaintext)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "WARNING: credential encryption disabled — running in plaintext mode\n")
		return nil
	}

	versionStr := os.Getenv(envKeyVersion)
	if versionStr == "" {
		fmt.Fprintf(os.Stderr, "FATAL: %s is set but %s is missing\n", envKeyring, envKeyVersion)
		os.Exit(1)
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %s must be an integer, got %q\n", envKeyVersion, versionStr)
		os.Exit(1)
	}

	kr, err := crypto.ParseKeyringEnv(raw, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: failed to parse %s: %v\n", envKeyring, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "INFO: credential encryption enabled, using key v%d, keyring contains versions: %v\n", version, kr.Versions())
	return kr
}

func IsPlaintextAllowed() bool {
	if os.Getenv(envAllowPlaintext) == "true" {
		return true
	}
	env := os.Getenv("AMBIENT_ENV")
	return env == "integration_testing" || env == "development"
}

func ValidateEncryptionStartup(db *gorm.DB, keyring *crypto.Keyring) {
	if keyring != nil {
		return
	}

	var count int64
	if err := db.Table("credentials").Where("token LIKE 'enc:v%'").Count(&count).Error; err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: could not check for encrypted tokens: %v\n", err)
		os.Exit(1)
	}

	if count > 0 {
		fmt.Fprintf(os.Stderr, "FATAL: %d encrypted credential tokens found but no keyring configured — set %s\n", count, envKeyring)
		os.Exit(1)
	}
}

func ValidateEncryptionStartupFromDAO(ctx context.Context, dao CredentialDao, keyring *crypto.Keyring) {
	if keyring != nil {
		return
	}

	all, err := dao.All(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: could not check for encrypted tokens: %v\n", err)
		os.Exit(1)
	}

	for _, c := range all {
		if c.Token != nil && crypto.IsEncrypted(*c.Token) {
			fmt.Fprintf(os.Stderr, "FATAL: encrypted credential tokens found but no keyring configured — set %s\n", envKeyring)
			os.Exit(1)
		}
	}
}
