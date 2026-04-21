package keypair

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/ambient-code/platform/components/ambient-control-plane/internal/kubeclient"
	"github.com/rs/zerolog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	SecretName    = "ambient-cp-token-keypair"
	privateKeyKey = "private.pem"
	publicKeyKey  = "public.pem"
	rsaKeyBits    = 4096
)

type KeyPair struct {
	PrivateKeyPEM []byte
	PublicKeyPEM  []byte
}

func EnsureKeypairSecret(ctx context.Context, kube *kubeclient.KubeClient, namespace string, logger zerolog.Logger) (*KeyPair, error) {
	existing, err := kube.GetSecret(ctx, namespace, SecretName)
	if err == nil {
		return keypairFromSecret(existing)
	}
	if !k8serrors.IsNotFound(err) {
		return nil, fmt.Errorf("checking for keypair secret: %w", err)
	}

	logger.Info().Str("namespace", namespace).Str("secret", SecretName).Msg("keypair secret not found, generating new RSA keypair")

	kp, err := generateKeypair()
	if err != nil {
		return nil, fmt.Errorf("generating RSA keypair: %w", err)
	}

	secret := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      SecretName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"app":                        "ambient-control-plane",
					"ambient-code.io/managed-by": "ambient-control-plane",
				},
			},
			"type": "Opaque",
			"data": map[string]interface{}{
				privateKeyKey: base64.StdEncoding.EncodeToString(kp.PrivateKeyPEM),
				publicKeyKey:  base64.StdEncoding.EncodeToString(kp.PublicKeyPEM),
			},
		},
	}

	if _, createErr := kube.CreateSecret(ctx, secret); createErr != nil {
		if !k8serrors.IsAlreadyExists(createErr) {
			return nil, fmt.Errorf("creating keypair secret: %w", createErr)
		}
		existing, err = kube.GetSecret(ctx, namespace, SecretName)
		if err != nil {
			return nil, fmt.Errorf("re-reading keypair secret after race: %w", err)
		}
		return keypairFromSecret(existing)
	}

	logger.Info().Str("namespace", namespace).Str("secret", SecretName).Msg("RSA keypair secret created")
	return kp, nil
}

func keypairFromSecret(secret *unstructured.Unstructured) (*KeyPair, error) {
	data, _, _ := unstructured.NestedMap(secret.Object, "data")

	privB64, ok := data[privateKeyKey].(string)
	if !ok || privB64 == "" {
		return nil, fmt.Errorf("keypair secret missing %q key", privateKeyKey)
	}
	pubB64, ok := data[publicKeyKey].(string)
	if !ok || pubB64 == "" {
		return nil, fmt.Errorf("keypair secret missing %q key", publicKeyKey)
	}

	privPEM, err := base64.StdEncoding.DecodeString(privB64)
	if err != nil {
		return nil, fmt.Errorf("decoding private key from secret: %w", err)
	}
	pubPEM, err := base64.StdEncoding.DecodeString(pubB64)
	if err != nil {
		return nil, fmt.Errorf("decoding public key from secret: %w", err)
	}

	return &KeyPair{PrivateKeyPEM: privPEM, PublicKeyPEM: pubPEM}, nil
}

func generateKeypair() (*KeyPair, error) {
	privKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, fmt.Errorf("generating RSA key: %w", err)
	}

	privPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privKey),
	})

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("marshaling public key: %w", err)
	}
	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubDER,
	})

	return &KeyPair{PrivateKeyPEM: privPEM, PublicKeyPEM: pubPEM}, nil
}

func ParsePrivateKey(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block for private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}
