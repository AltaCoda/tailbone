package utils

import (
	"context"
)

// CloudConnector interface defines the methods for cloud operations
type CloudConnector interface {
	Upload(ctx context.Context, bucket, key string, data []byte) error
	Download(ctx context.Context, bucket, key string) ([]byte, error)
	GetBucketAndKeyPath(ctx context.Context) (string, string, error)
}
