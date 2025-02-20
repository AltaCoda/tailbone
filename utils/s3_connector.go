package utils

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/spf13/viper"
)

// s3Connector implements the CloudConnector interface
type s3Connector struct {
	client *s3.Client
}

// NewS3Connector creates a new S3Connector instance
func NewS3Connector(ctx context.Context) (CloudConnector, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := s3.NewFromConfig(cfg)
	return &s3Connector{client: client}, nil
}

// Upload uploads data to a cloud bucket with the specified content type
func (s *s3Connector) Upload(ctx context.Context, bucket, key string, data []byte) error {
	if bucket == "" {
		return fmt.Errorf("bucket name is required")
	}

	// Upload to S3
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}

	return nil
}

func (s *s3Connector) GetBucketAndKeyPath(ctx context.Context) (string, string, error) {
	if viper.GetString("keys.bucket") == "" {
		return "", "", fmt.Errorf("keys.bucket is required")
	}

	keyPath := ".well-known/jwks.json"
	if viper.GetString("keys.keyPath") != "" {
		keyPath = viper.GetString("keys.keyPath")
	}

	return viper.GetString("keys.bucket"), keyPath, nil
}

func (s *s3Connector) Download(ctx context.Context, bucket, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}

	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

var _ CloudConnector = &s3Connector{}
