package projectKeys

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/openshift-online/rh-trex-ai/pkg/api"
)

const (
	keyByteLength = 32
	keyPrefixLen  = 8
	bcryptCost    = 12
)

type ProjectKey struct {
	api.Meta
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	KeyHash    string     `json:"-"`
	ProjectId  *string    `json:"project_id"`
	ExpiresAt  *time.Time `json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`

	PlaintextKey string `json:"-" gorm:"-"`
}

type ProjectKeyList []*ProjectKey
type ProjectKeyIndex map[string]*ProjectKey

func (l ProjectKeyList) Index() ProjectKeyIndex {
	index := ProjectKeyIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func generatePlaintextKey() (string, error) {
	b := make([]byte, keyByteLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return "ak_" + base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

func (d *ProjectKey) BeforeCreate(tx *gorm.DB) error {
	d.ID = api.NewID()

	if d.Name == "" {
		return fmt.Errorf("name is required")
	}

	plaintext, err := generatePlaintextKey()
	if err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcryptCost)
	if err != nil {
		return fmt.Errorf("failed to hash key: %w", err)
	}

	d.PlaintextKey = plaintext
	d.KeyPrefix = plaintext[:keyPrefixLen]
	d.KeyHash = string(hash)

	return nil
}
