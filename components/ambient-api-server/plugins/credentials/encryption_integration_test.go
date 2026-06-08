package credentials_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/pkg/crypto"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/credentials"
	"github.com/ambient-code/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func testKeyring(t *testing.T) *crypto.Keyring {
	t.Helper()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 42)
	}
	kr, err := crypto.NewKeyring(map[string]string{
		"1": base64.StdEncoding.EncodeToString(key),
	}, 1)
	if err != nil {
		t.Fatalf("NewKeyring: %v", err)
	}
	return kr
}

func newEncryptedCredential(t *testing.T, name string, token string) *credentials.Credential {
	t.Helper()
	kr := testKeyring(t)
	svc := credentials.NewCredentialService(
		nil,
		credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory),
		nil,
		kr,
	)

	// Advisory locks need the lock factory; use a simpler approach via DAO directly
	// since we just need to test encrypt/decrypt roundtrip through the service
	credential := &credentials.Credential{
		Name:     name,
		Provider: "github",
		Token:    &token,
	}

	created, err := svc.Create(context.Background(), credential)
	if err != nil {
		t.Fatalf("Create encrypted credential: %v", err)
	}
	return created
}

func TestEncryptedCredentialRoundtrip(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	plaintext := "ghp_superSecretToken123"
	created := newEncryptedCredential(t, h.NewID(), plaintext)

	// Read raw DB value to verify it's ciphertext
	dao := credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory)
	raw, err := dao.Get(context.Background(), created.ID)
	Expect(err).NotTo(HaveOccurred())
	Expect(raw.Token).NotTo(BeNil())
	Expect(*raw.Token).To(HavePrefix("enc:v1:"), "DB should contain encrypted token")
	Expect(*raw.Token).NotTo(Equal(plaintext), "DB must not contain plaintext")

	// Read through service (which decrypts)
	kr := testKeyring(t)
	svc := credentials.NewCredentialService(
		nil,
		dao,
		nil,
		kr,
	)
	decrypted, svcErr := svc.Get(context.Background(), created.ID)
	Expect(svcErr).NotTo(HaveOccurred())
	Expect(*decrypted.Token).To(Equal(plaintext), "Service.Get must return decrypted plaintext")
}

func TestEncryptedCredentialViaAPI(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	credentialInput := openapi.Credential{
		ProjectId: openapi.PtrString(testProjectID),
		Name:      h.NewID(),
		Provider:  "github",
		Token:     openapi.PtrString("ghp_apiTestToken456"),
	}

	credentialOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsPost(ctx, testProjectID).Credential(credentialInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(credentialOutput.Token).To(BeNil(), "POST response must never return the token value")

	// Fetch token via API
	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, restyErr := resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		Get(h.RestURL(fmt.Sprintf("/credentials/%s/token", *credentialOutput.Id)))
	Expect(restyErr).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusOK))

	// API returns plaintext (encryption is transparent)
	var tokenResponse struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(restyResp.Body(), &tokenResponse)
	Expect(err).NotTo(HaveOccurred())
	Expect(tokenResponse.Token).To(Equal(*credentialInput.Token), "GET /token must return the original plaintext token")
	Expect(tokenResponse.Token).NotTo(HavePrefix("enc:v1:"), "API must not return ciphertext")
}

func TestPlaintextTokenPassthrough(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	// Create a credential WITHOUT encryption (nil keyring = plaintext mode)
	svc := credentials.NewCredentialService(
		nil,
		credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory),
		nil,
		nil,
	)

	plaintext := "ghp_plaintext_no_encryption"
	credential := &credentials.Credential{
		Name:     h.NewID(),
		Provider: "github",
		Token:    &plaintext,
	}

	created, err := svc.Create(context.Background(), credential)
	Expect(err).NotTo(HaveOccurred())

	// Read through a service WITH encryption enabled — should return plaintext as-is
	kr := testKeyring(t)
	encSvc := credentials.NewCredentialService(
		nil,
		credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory),
		nil,
		kr,
	)

	got, svcErr := encSvc.Get(context.Background(), created.ID)
	Expect(svcErr).NotTo(HaveOccurred())
	Expect(*got.Token).To(Equal(plaintext), "plaintext tokens must pass through without decryption")
}

func TestTokenVersionDetection(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	created := newEncryptedCredential(t, h.NewID(), "secret123")

	dao := credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory)
	raw, err := dao.Get(context.Background(), created.ID)
	Expect(err).NotTo(HaveOccurred())

	Expect(crypto.IsEncrypted(*raw.Token)).To(BeTrue())
	v, ok := crypto.TokenVersion(*raw.Token)
	Expect(ok).To(BeTrue())
	Expect(v).To(Equal(1))
}

func TestAADPreventsRowSwap(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	cred1 := newEncryptedCredential(t, h.NewID(), "secret_A")
	cred2 := newEncryptedCredential(t, h.NewID(), "secret_B")

	dao := credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory)
	raw1, err := dao.Get(context.Background(), cred1.ID)
	Expect(err).NotTo(HaveOccurred())
	Expect(raw1.Token).NotTo(BeNil())
	raw2, err := dao.Get(context.Background(), cred2.ID)
	Expect(err).NotTo(HaveOccurred())
	Expect(raw2.Token).NotTo(BeNil())

	// Swap ciphertexts
	kr := testKeyring(t)
	_, err1 := kr.Decrypt(*raw1.Token, cred2.ID)
	Expect(err1).To(HaveOccurred(), "decrypting cred1's token with cred2's ID must fail (AAD mismatch)")

	_, err2 := kr.Decrypt(*raw2.Token, cred1.ID)
	Expect(err2).To(HaveOccurred(), "decrypting cred2's token with cred1's ID must fail (AAD mismatch)")

	// But correct IDs work
	dec1, err := kr.Decrypt(*raw1.Token, cred1.ID)
	Expect(err).NotTo(HaveOccurred())
	Expect(dec1).To(Equal("secret_A"))

	dec2, err := kr.Decrypt(*raw2.Token, cred2.ID)
	Expect(err).NotTo(HaveOccurred())
	Expect(dec2).To(Equal("secret_B"))
}

func TestEncryptedLargeKubeconfig(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	kubeconfig := strings.Repeat("apiVersion: v1\nkind: Config\nclusters:\n", 100)
	created := newEncryptedCredential(t, h.NewID(), kubeconfig)

	kr := testKeyring(t)
	dao := credentials.NewCredentialDao(&environments.Environment().Database.SessionFactory)

	svc := credentials.NewCredentialService(nil, dao, nil, kr)
	got, svcErr := svc.Get(context.Background(), created.ID)
	Expect(svcErr).NotTo(HaveOccurred())
	Expect(*got.Token).To(Equal(kubeconfig))
}
