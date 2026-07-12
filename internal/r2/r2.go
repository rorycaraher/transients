// Package r2 wraps the S3-compatible API for Cloudflare R2: presigned
// PUT/GET URLs for browser upload/playback, and object download/delete for
// the ingest pipeline.
package r2

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// ErrNotFound is returned by Head when the object doesn't exist in R2 (e.g.
// deleted after its event notification was already enqueued).
var ErrNotFound = errors.New("object not found")

type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	bucket  string
}

func New(accountID, accessKeyID, secretAccessKey, bucket string) *Client {
	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	cli := s3.New(s3.Options{
		Region:       "auto",
		BaseEndpoint: aws.String(endpoint),
		Credentials:  credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		UsePathStyle: true,
	})

	return &Client{
		s3:      cli,
		presign: s3.NewPresignClient(cli),
		bucket:  bucket,
	}
}

// PresignPut returns a presigned URL the browser can PUT the file to directly.
func (c *Client) PresignPut(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign put %s: %w", key, err)
	}
	return req.URL, nil
}

// PresignGet returns a presigned URL for streaming playback.
func (c *Client) PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presign get %s: %w", key, err)
	}
	return req.URL, nil
}

type ObjectMeta struct {
	ContentType string
	SizeBytes   int64
}

func (c *Client) Head(ctx context.Context, key string) (*ObjectMeta, error) {
	out, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var respErr *smithyhttp.ResponseError
		if errors.As(err, &respErr) && respErr.HTTPStatusCode() == 404 {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("head %s: %w", key, err)
	}
	meta := &ObjectMeta{}
	if out.ContentType != nil {
		meta.ContentType = *out.ContentType
	}
	if out.ContentLength != nil {
		meta.SizeBytes = *out.ContentLength
	}
	return meta, nil
}

// DownloadToTempFile fetches the object and writes it to a temp file,
// returning its path. The caller is responsible for removing it.
func (c *Client) DownloadToTempFile(ctx context.Context, key string) (string, error) {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return "", fmt.Errorf("get object %s: %w", key, err)
	}
	defer out.Body.Close()

	f, err := os.CreateTemp("", "transients-ingest-*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, out.Body); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("write temp file: %w", err)
	}
	return f.Name(), nil
}

func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}
