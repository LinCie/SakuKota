package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"rantau-go/internal/config"
)

// S3 is an ObjectStore backed by an S3-compatible API such as Garage.
type S3 struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

var _ ObjectStore = (*S3)(nil)

func NewS3(cfg config.StorageConfig) *S3 {
	options := func(endpoint string) s3.Options {
		return s3.Options{
			BaseEndpoint: aws.String(endpoint),
			Region:       cfg.Region,
			Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
			UsePathStyle: true,
		}
	}

	return &S3{
		client:    s3.New(options(cfg.Endpoint)),
		presigner: s3.NewPresignClient(s3.New(options(cfg.PublicEndpoint))),
		bucket:    cfg.Bucket,
	}
}

// -----------------------------------------------------------------------------
// S3
// -----------------------------------------------------------------------------

// PresignPut signs the upload for cfg.PublicEndpoint, the host clients send it to.
func (s *S3) PresignPut(ctx context.Context, key, contentType string, ttl time.Duration) (PresignedUpload, error) {
	expiresAt := time.Now().Add(ttl)

	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return PresignedUpload{}, err
	}

	// Clients cannot set Host; it comes from the URL.
	headers := make(map[string]string, len(req.SignedHeader))
	for name, values := range req.SignedHeader {
		if name != "Host" {
			headers[name] = strings.Join(values, ",")
		}
	}

	return PresignedUpload{
		URL:       req.URL,
		Method:    req.Method,
		Headers:   headers,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *S3) Stat(ctx context.Context, key string) (ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return ObjectInfo{}, ErrObjectNotFound
	}
	if err != nil {
		return ObjectInfo{}, err
	}

	return ObjectInfo{
		Size:        aws.ToInt64(out.ContentLength),
		ContentType: aws.ToString(out.ContentType),
	}, nil
}

// Copy copies srcKey to dstKey in the bucket. srcKey is not URL-escaped, so it
// must be made of URL-safe characters, as the keys Images creates are.
func (s *S3) Copy(ctx context.Context, srcKey, dstKey string) error {
	_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(dstKey),
		CopySource: aws.String(s.bucket + "/" + srcKey),
	})
	return err
}

// Delete removes the object. Deleting a missing object is not an error.
func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}
