// Package storage provides S3-compatible object storage operations for file uploads.
package storage

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Client wraps the MinIO client for S3-compatible storage operations.
type S3Client struct {
	client *minio.Client
	bucket string
}

// S3Config holds the configuration for connecting to S3-compatible storage.
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

// S3FileInfo represents metadata about a file stored in S3.
type S3FileInfo struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	LastModified string `json:"lastModified"`
	ContentType  string `json:"contentType,omitempty"`
}

// LoadS3ConfigFromEnv reads S3 configuration from environment variables.
func LoadS3ConfigFromEnv() (*S3Config, error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	bucket := os.Getenv("S3_BUCKET")
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	if endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("S3_ENDPOINT and S3_BUCKET must be set")
	}
	if accessKey == "" || secretKey == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
	}

	// Determine SSL from endpoint scheme
	useSSL := strings.HasPrefix(endpoint, "https://")

	// Strip scheme for MinIO client (it adds its own)
	endpoint = strings.TrimPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(endpoint, "http://")

	return &S3Config{
		Endpoint:  endpoint,
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    useSSL,
	}, nil
}

// NewS3Client creates a new S3Client from the given config.
func NewS3Client(cfg *S3Config) (*S3Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &S3Client{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// PutObject uploads a file to the given S3 key.
func (s *S3Client) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{}
	if contentType != "" {
		opts.ContentType = contentType
	}

	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, opts)
	if err != nil {
		return fmt.Errorf("failed to upload to S3 key %q: %w", key, err)
	}

	log.Printf("S3: uploaded %s (%d bytes)", key, size)
	return nil
}

// ListObjects lists all objects under a given prefix.
func (s *S3Client) ListObjects(ctx context.Context, prefix string) ([]S3FileInfo, error) {
	var files []S3FileInfo

	objectCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return nil, fmt.Errorf("failed to list S3 objects: %w", obj.Err)
		}

		// Strip prefix from key for display
		relKey := strings.TrimPrefix(obj.Key, prefix)
		if relKey == "" {
			continue
		}

		files = append(files, S3FileInfo{
			Key:          relKey,
			Size:         obj.Size,
			LastModified: obj.LastModified.UTC().Format("2006-01-02T15:04:05Z"),
			ContentType:  obj.ContentType,
		})
	}

	return files, nil
}

// DeleteObject deletes a single object from S3.
func (s *S3Client) DeleteObject(ctx context.Context, key string) error {
	err := s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete S3 key %q: %w", key, err)
	}

	log.Printf("S3: deleted %s", key)
	return nil
}

// ObjectExists checks if an object exists at the given key.
func (s *S3Client) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		return false, fmt.Errorf("failed to stat S3 key %q: %w", key, err)
	}
	return true, nil
}
