package keypair

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"testing"

	"github.com/rs/zerolog"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
)

func newFakeKubeClient(objects ...runtime.Object) *kubeclient.KubeClient {
	scheme := runtime.NewScheme()
	dynClient := fake.NewSimpleDynamicClient(scheme, objects...)
	return kubeclient.NewFromDynamic(dynClient, zerolog.Nop())
}

func TestGenerateKeypair(t *testing.T) {
	kp, err := generateKeypair()
	if err != nil {
		t.Fatalf("generateKeypair() error: %v", err)
	}
	if len(kp.PrivateKeyPEM) == 0 {
		t.Error("PrivateKeyPEM is empty")
	}
	if len(kp.PublicKeyPEM) == 0 {
		t.Error("PublicKeyPEM is empty")
	}
}

func TestParsePrivateKey(t *testing.T) {
	kp, err := generateKeypair()
	if err != nil {
		t.Fatalf("generateKeypair() error: %v", err)
	}
	privKey, err := ParsePrivateKey(kp.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("ParsePrivateKey() error: %v", err)
	}
	if privKey == nil {
		t.Fatal("ParsePrivateKey() returned nil")
	}
	if _, ok := interface{}(privKey).(*rsa.PrivateKey); !ok {
		t.Error("parsed key is not *rsa.PrivateKey")
	}
}

func TestParsePrivateKey_InvalidPEM(t *testing.T) {
	_, err := ParsePrivateKey([]byte("not a pem block"))
	if err == nil {
		t.Error("expected error for invalid PEM, got nil")
	}
}

func TestKeypairFromSecret_MissingPrivateKey(t *testing.T) {
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata":   map[string]interface{}{"name": SecretName, "namespace": "test"},
			"data": map[string]interface{}{
				publicKeyKey: base64.StdEncoding.EncodeToString([]byte("pub")),
			},
		},
	}
	_, err := keypairFromSecret(secret)
	if err == nil {
		t.Error("expected error for missing private key, got nil")
	}
}

func TestKeypairFromSecret_MissingPublicKey(t *testing.T) {
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata":   map[string]interface{}{"name": SecretName, "namespace": "test"},
			"data": map[string]interface{}{
				privateKeyKey: base64.StdEncoding.EncodeToString([]byte("priv")),
			},
		},
	}
	_, err := keypairFromSecret(secret)
	if err == nil {
		t.Error("expected error for missing public key, got nil")
	}
}

func TestKeypairFromSecret_ValidSecret(t *testing.T) {
	kp, err := generateKeypair()
	if err != nil {
		t.Fatalf("generateKeypair() error: %v", err)
	}
	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata":   map[string]interface{}{"name": SecretName, "namespace": "test"},
			"data": map[string]interface{}{
				privateKeyKey: base64.StdEncoding.EncodeToString(kp.PrivateKeyPEM),
				publicKeyKey:  base64.StdEncoding.EncodeToString(kp.PublicKeyPEM),
			},
		},
	}
	got, err := keypairFromSecret(secret)
	if err != nil {
		t.Fatalf("keypairFromSecret() error: %v", err)
	}
	if string(got.PrivateKeyPEM) != string(kp.PrivateKeyPEM) {
		t.Error("PrivateKeyPEM mismatch")
	}
	if string(got.PublicKeyPEM) != string(kp.PublicKeyPEM) {
		t.Error("PublicKeyPEM mismatch")
	}
}

func TestEnsureKeypairSecret_CreatesWhenMissing(t *testing.T) {
	kube := newFakeKubeClient()
	ctx := context.Background()

	kp, err := EnsureKeypairSecret(ctx, kube, "test-ns", zerolog.Nop())
	if err != nil {
		t.Fatalf("EnsureKeypairSecret() error: %v", err)
	}
	if len(kp.PrivateKeyPEM) == 0 || len(kp.PublicKeyPEM) == 0 {
		t.Error("returned keypair has empty PEM fields")
	}

	privKey, err := ParsePrivateKey(kp.PrivateKeyPEM)
	if err != nil {
		t.Fatalf("generated private key is not parseable: %v", err)
	}
	if privKey.N.BitLen() != rsaKeyBits {
		t.Errorf("key size: got %d, want %d", privKey.N.BitLen(), rsaKeyBits)
	}
}

func TestEnsureKeypairSecret_ReturnsExistingWhenPresent(t *testing.T) {
	ctx := context.Background()
	kube := newFakeKubeClient()

	first, err := EnsureKeypairSecret(ctx, kube, "test-ns", zerolog.Nop())
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	second, err := EnsureKeypairSecret(ctx, kube, "test-ns", zerolog.Nop())
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if string(first.PrivateKeyPEM) != string(second.PrivateKeyPEM) {
		t.Error("second call returned different private key — should reuse existing Secret")
	}
}
