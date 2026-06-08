package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/crypto"
	"github.com/openshift-online/rh-trex-ai/pkg/config"
	"github.com/openshift-online/rh-trex-ai/pkg/db/db_session"
	"gorm.io/gorm"
)

type credentialRow struct {
	ID    string  `gorm:"column:id"`
	Token *string `gorm:"column:token"`
}

func (credentialRow) TableName() string { return "credentials" }

func NewEncryptCredentialsCommand() *cobra.Command {
	dbConfig := config.NewDatabaseConfig()
	var decrypt bool
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "encrypt-credentials",
		Short: "Encrypt or re-encrypt credential tokens at rest",
		Long:  "Bulk encrypt plaintext tokens, re-encrypt tokens to the current key version, or decrypt all tokens to plaintext (--decrypt).",
		Run: func(cmd *cobra.Command, args []string) {
			if err := dbConfig.ReadFiles(); err != nil {
				glog.Fatal(err)
			}

			keyringRaw := os.Getenv("CREDENTIAL_ENCRYPTION_KEYRING")
			versionStr := os.Getenv("CREDENTIAL_ENCRYPTION_KEY_VERSION")

			if keyringRaw == "" || versionStr == "" {
				glog.Fatal("CREDENTIAL_ENCRYPTION_KEYRING and CREDENTIAL_ENCRYPTION_KEY_VERSION must be set")
			}

			activeVersion, err := strconv.Atoi(versionStr)
			if err != nil {
				glog.Fatalf("CREDENTIAL_ENCRYPTION_KEY_VERSION must be an integer: %v", err)
			}

			keyring, err := crypto.ParseKeyringEnv(keyringRaw, activeVersion)
			if err != nil {
				glog.Fatalf("Failed to parse keyring: %v", err)
			}

			connection := db_session.NewProdFactory(dbConfig)
			db := connection.New(context.Background())

			if decrypt {
				runDecrypt(db, keyring, dryRun)
			} else {
				runEncrypt(db, keyring, activeVersion, dryRun)
			}
		},
	}

	cmd.Flags().BoolVar(&decrypt, "decrypt", false, "Decrypt all tokens to plaintext (rollback)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Report what would be done without modifying data")

	dbConfig.AddFlags(cmd.PersistentFlags())
	cmd.PersistentFlags().AddGoFlagSet(flag.CommandLine)
	return cmd
}

func runEncrypt(db *gorm.DB, keyring *crypto.Keyring, activeVersion int, dryRun bool) {
	var rows []credentialRow
	if err := db.Where("token IS NOT NULL AND token != ''").Find(&rows).Error; err != nil {
		glog.Fatalf("Failed to query credentials: %v", err)
	}

	var plaintextCandidates, reencryptCandidates, current int
	var plaintextSuccess, reencryptSuccess, failed int
	var failedIDs []string

	for _, row := range rows {
		if row.Token == nil || *row.Token == "" {
			continue
		}

		token := *row.Token

		if !crypto.IsEncrypted(token) {
			plaintextCandidates++
			if dryRun {
				continue
			}
			encrypted, err := keyring.Encrypt(token, row.ID)
			if err != nil {
				failed++
				failedIDs = append(failedIDs, row.ID)
				fmt.Fprintf(os.Stderr, "ERROR: encrypt %s: %v\n", row.ID, err)
				continue
			}
			if err := db.Model(&credentialRow{}).Where("id = ?", row.ID).Update("token", encrypted).Error; err != nil {
				failed++
				failedIDs = append(failedIDs, row.ID)
				fmt.Fprintf(os.Stderr, "ERROR: update %s: %v\n", row.ID, err)
				continue
			}
			plaintextSuccess++
		} else {
			v, _ := crypto.TokenVersion(token)
			if v == activeVersion {
				current++
				continue
			}
			reencryptCandidates++
			if dryRun {
				continue
			}
			decrypted, err := keyring.Decrypt(token, row.ID)
			if err != nil {
				failed++
				failedIDs = append(failedIDs, row.ID)
				fmt.Fprintf(os.Stderr, "ERROR: decrypt %s for re-encryption: %v\n", row.ID, err)
				continue
			}
			encrypted, err := keyring.Encrypt(decrypted, row.ID)
			if err != nil {
				failed++
				failedIDs = append(failedIDs, row.ID)
				fmt.Fprintf(os.Stderr, "ERROR: re-encrypt %s: %v\n", row.ID, err)
				continue
			}
			if err := db.Model(&credentialRow{}).Where("id = ?", row.ID).Update("token", encrypted).Error; err != nil {
				failed++
				failedIDs = append(failedIDs, row.ID)
				fmt.Fprintf(os.Stderr, "ERROR: update %s: %v\n", row.ID, err)
				continue
			}
			reencryptSuccess++
		}
	}

	if dryRun {
		fmt.Printf("Would encrypt: %d plaintext, Would re-encrypt: %d (→ v%d), Already current: %d\n", plaintextCandidates, reencryptCandidates, activeVersion, current)
		return
	}

	if plaintextSuccess+reencryptSuccess == 0 && failed == 0 {
		fmt.Println("0 credentials need encryption. All up to date.")
		return
	}

	if plaintextSuccess > 0 {
		fmt.Printf("%d credentials encrypted (plaintext → v%d)\n", plaintextSuccess, activeVersion)
	}
	if reencryptSuccess > 0 {
		fmt.Printf("%d credentials re-encrypted to v%d\n", reencryptSuccess, activeVersion)
	}
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d credentials failed: %v\n", failed, failedIDs)
		os.Exit(1)
	}
}

func runDecrypt(db *gorm.DB, keyring *crypto.Keyring, dryRun bool) {
	var rows []credentialRow
	if err := db.Where("token IS NOT NULL AND token != ''").Find(&rows).Error; err != nil {
		glog.Fatalf("Failed to query credentials: %v", err)
	}

	var decrypted, skipped, failed int
	var failedIDs []string

	for _, row := range rows {
		if row.Token == nil || *row.Token == "" {
			continue
		}

		token := *row.Token
		if !crypto.IsEncrypted(token) {
			skipped++
			continue
		}

		decrypted++
		if dryRun {
			continue
		}

		plaintext, err := keyring.Decrypt(token, row.ID)
		if err != nil {
			failed++
			failedIDs = append(failedIDs, row.ID)
			fmt.Fprintf(os.Stderr, "ERROR: decrypt %s: %v\n", row.ID, err)
			continue
		}
		if err := db.Model(&credentialRow{}).Where("id = ?", row.ID).Update("token", plaintext).Error; err != nil {
			failed++
			failedIDs = append(failedIDs, row.ID)
			fmt.Fprintf(os.Stderr, "ERROR: update %s: %v\n", row.ID, err)
			continue
		}
	}

	if dryRun {
		fmt.Printf("Would decrypt: %d encrypted, Already plaintext: %d\n", decrypted, skipped)
		return
	}

	if decrypted == 0 {
		fmt.Println("0 credentials need decryption. All plaintext.")
		return
	}

	fmt.Printf("%d credentials decrypted to plaintext\n", decrypted-failed)
	if failed > 0 {
		fmt.Fprintf(os.Stderr, "%d credentials failed: %v\n", failed, failedIDs)
		os.Exit(1)
	}
}
