package services

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"tixora/internal/config"
)

// IStorageService talks to the object storage provider (R2, S3-compatible).
type IStorageService interface {
	PresignPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error)
}

type StorageService struct {
	presignClient *s3.PresignClient
	bucket        string
}

// NewStorageService builds an S3-compatible client pointed at Cloudflare R2.
func NewStorageService(cfg *config.Config) IStorageService {
	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.R2Endpoint),
		Region:       cfg.R2Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.R2AccessKey, cfg.R2SecretKey, ""),
		UsePathStyle: true,
	})

	return &StorageService{
		presignClient: s3.NewPresignClient(client),
		bucket:        cfg.R2Bucket,
	}
}

// PresignPutURL returns a time-limited URL the client can PUT the object
// bytes to directly, without the request ever passing through our server.
func (s *StorageService) PresignPutURL(ctx context.Context, objectKey, contentType string, expiry time.Duration) (string, error) {
	req, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(objectKey),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("failed to presign upload url: %w", err)
	}

	return req.URL, nil
}
