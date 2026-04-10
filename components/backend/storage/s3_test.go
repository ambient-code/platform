package storage

import (
	"os"
	"testing"
)

func TestLoadS3ConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		wantErr bool
		check   func(t *testing.T, cfg *S3Config)
	}{
		{
			name: "all env vars set with http endpoint",
			envVars: map[string]string{
				"S3_ENDPOINT":           "http://minio.svc:9000",
				"S3_BUCKET":             "test-bucket",
				"AWS_ACCESS_KEY_ID":     "testkey",
				"AWS_SECRET_ACCESS_KEY": "testsecret",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *S3Config) {
				if cfg.Endpoint != "minio.svc:9000" {
					t.Errorf("expected endpoint 'minio.svc:9000', got %q", cfg.Endpoint)
				}
				if cfg.Bucket != "test-bucket" {
					t.Errorf("expected bucket 'test-bucket', got %q", cfg.Bucket)
				}
				if cfg.UseSSL {
					t.Error("expected UseSSL=false for http endpoint")
				}
			},
		},
		{
			name: "https endpoint enables SSL",
			envVars: map[string]string{
				"S3_ENDPOINT":           "https://s3.amazonaws.com",
				"S3_BUCKET":             "prod-bucket",
				"AWS_ACCESS_KEY_ID":     "key",
				"AWS_SECRET_ACCESS_KEY": "secret",
			},
			wantErr: false,
			check: func(t *testing.T, cfg *S3Config) {
				if cfg.Endpoint != "s3.amazonaws.com" {
					t.Errorf("expected endpoint 's3.amazonaws.com', got %q", cfg.Endpoint)
				}
				if !cfg.UseSSL {
					t.Error("expected UseSSL=true for https endpoint")
				}
			},
		},
		{
			name: "missing endpoint and bucket",
			envVars: map[string]string{
				"AWS_ACCESS_KEY_ID":     "key",
				"AWS_SECRET_ACCESS_KEY": "secret",
			},
			wantErr: true,
		},
		{
			name: "missing credentials",
			envVars: map[string]string{
				"S3_ENDPOINT": "http://minio:9000",
				"S3_BUCKET":   "bucket",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear env vars
			for _, key := range []string{"S3_ENDPOINT", "S3_BUCKET", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
				os.Unsetenv(key)
			}
			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			cfg, err := LoadS3ConfigFromEnv()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
