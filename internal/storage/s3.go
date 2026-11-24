package storage

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"sakukota-go/internal/config"
)

// S3 is an ObjectStore backed by an S3-compatible API such as Garage.
type S3 struct {
	client    *s3.Client
	presigner *s3.PresignClient
	bucket    string
}

var _ ObjectStore = (*S3)(nil)

// NewS3 creates a store client from cfg. Temporary upload links are signed for the
// public endpoint, which is the host clients reach.
func NewS3(cfg config.StorageConfig) *S3 {
	opts := s3.Options{
		BaseEndpoint: aws.String(cfg.Endpoint),
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
	}
	client := s3.New(opts)

	opts.BaseEndpoint = aws.String(cfg.PublicEndpoint)
	presigner := s3.NewPresignClient(s3.New(opts))

	return &S3{client: client, presigner: presigner, bucket: cfg.Bucket}
}

// -----------------------------------------------------------------------------
// S3
// -----------------------------------------------------------------------------

// PresignPut returns a temporary link that uploads key, signed for the public endpoint clients use.
func (s *S3) PresignPut(ctx context.Context, key, contentType string) (Upload, error) {
	expiresAt := time.Now().Add(UploadTTL)

	req, err := s.presigner.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}, s3.WithPresignExpires(UploadTTL))
	if err != nil {
		return Upload{}, err
	}

	// Clients cannot set Host; it comes from the URL.
	headers := make(map[string]string, len(req.SignedHeader))
	for name, values := range req.SignedHeader {
		if name != "Host" {
			headers[name] = strings.Join(values, ",")
		}
	}

	return Upload{
		Key:       key,
		URL:       req.URL,
		Method:    req.Method,
		Headers:   headers,
		ExpiresAt: expiresAt,
	}, nil
}

// Put stores body under key with contentType, replacing any object already there.
func (s *S3) Put(ctx context.Context, key, contentType string, body []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
		Body:        bytes.NewReader(body),
	})
	return err
}

// Stat returns the object's size and type, or ErrObjectNotFound if it is missing.
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
// must be made of URL-safe characters, as the keys MediaStore creates are.
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
