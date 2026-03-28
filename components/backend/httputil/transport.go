// Package httputil provides a shared HTTP transport with custom CA support.
package httputil

import (
	"crypto/tls"
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

var (
	sharedTransport *http.Transport
	once            sync.Once
)

// Transport returns a shared http.Transport configured with custom CA
// certificates when available. It appends certificates from the file
// specified by the CUSTOM_CA_BUNDLE environment variable to the system
// certificate pool. The transport is created once and reused for all
// callers so that idle connections are shared.
func Transport() *http.Transport {
	once.Do(func() {
		sharedTransport = buildTransport()
	})
	return sharedTransport
}

// NewClient returns an *http.Client that uses the shared transport with
// custom CA support and the given timeout.
func NewClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: Transport(),
	}
}

func buildTransport() *http.Transport {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	if caBundle := os.Getenv("CUSTOM_CA_BUNDLE"); caBundle != "" {
		if pool := loadCustomCAs(caBundle); pool != nil {
			tlsConfig.RootCAs = pool
		}
	}

	// Mirror net/http.DefaultTransport settings so we preserve proxy,
	// HTTP/2, dial timeouts, and keep-alive behaviour.
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		TLSClientConfig:       tlsConfig,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

// loadCustomCAs returns a cert pool that combines the system CAs with the
// PEM certificates in bundlePath. It returns nil if the extra certs cannot
// be loaded, so callers can fall back to Go's default behaviour.
func loadCustomCAs(bundlePath string) *x509.CertPool {
	pem, err := os.ReadFile(bundlePath)
	if err != nil {
		log.Printf("WARNING: failed to read CUSTOM_CA_BUNDLE (%s): %v", bundlePath, err)
		return nil
	}

	pool, err := x509.SystemCertPool()
	if err != nil {
		log.Printf("WARNING: failed to load system cert pool, creating empty pool: %v", err)
		pool = x509.NewCertPool()
	}

	if !pool.AppendCertsFromPEM(pem) {
		log.Printf("WARNING: CUSTOM_CA_BUNDLE (%s) contained no valid PEM certificates", bundlePath)
		return nil
	}

	log.Printf("Loaded custom CA certificates from %s", bundlePath)
	return pool
}
