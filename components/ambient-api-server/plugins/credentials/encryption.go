package credentials

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/crypto"
)

const (
	envKeyring        = "CREDENTIAL_ENCRYPTION_KEYRING"
	envKeyVersion     = "CREDENTIAL_ENCRYPTION_KEY_VERSION"
	envAllowPlaintext = "CREDENTIAL_ENCRYPTION_ALLOW_PLAINTEXT"
)

func LoadKeyring() *crypto.Keyring {
	raw := os.Getenv(envKeyring)
	if raw == "" {
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
	return os.Getenv(envAllowPlaintext) == "true"
}
