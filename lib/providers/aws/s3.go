package aws

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/input-output-hk/catalyst-forge/lib/tools/fs"
	"github.com/input-output-hk/catalyst-forge/lib/tools/walker"
)

//go:generate go run github.com/matryer/moq@latest -skip-ensure --pkg mocks -out mocks/s3.go . AWSS3Client

// AWSS3Client is an interface for an AWS S3 client.
type AWSS3Client interface {
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3Client is a client for interacting with AWS S3.
type S3Client struct {
	client AWSS3Client
	logger *slog.Logger
	walker walker.Walker
}

// UploadFile uploads a file to S3.
func (c *S3Client) UploadFile(bucket, key string, body io.Reader, contentType string) error {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
	}

	_, err := c.client.PutObject(context.Background(), input)
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	c.logger.Debug("Successfully uploaded file to S3", "bucket", bucket, "key", key)
	return nil
}

// DeleteFile deletes a file from S3.
func (c *S3Client) DeleteFile(bucket, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err := c.client.DeleteObject(context.Background(), input)
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	c.logger.Debug("Successfully deleted file from S3", "bucket", bucket, "key", key)
	return nil
}

// DeleteDirectory deletes all objects in a directory (prefix) from S3.
// exclude is a list of regexes to exclude from deletion.
func (c *S3Client) DeleteDirectory(bucket, prefix string, exclude []string) error {
	listInput := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	}

	listOutput, err := c.client.ListObjectsV2(context.Background(), listInput)
	if err != nil {
		return fmt.Errorf("failed to list objects in S3: %w", err)
	}

	for _, obj := range listOutput.Contents {
		skip := false
		for _, pat := range exclude {
			if regexp.MustCompile(pat).MatchString(*obj.Key) {
				skip = true
				break
			}
		}
		if skip {
			c.logger.Debug("Skipping delete (excluded)", "key", *obj.Key)
			continue
		}

		if err := c.DeleteFile(bucket, *obj.Key); err != nil {
			return fmt.Errorf("failed to delete object %s: %w", *obj.Key, err)
		}
	}

	c.logger.Debug("Successfully deleted directory from S3", "bucket", bucket, "prefix", prefix)
	return nil
}

func (c *S3Client) ListImmediateChildren(bucket, prefix string) ([]string, error) {
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	var children []string
	seen := make(map[string]struct{})

	input := &s3.ListObjectsV2Input{
		Bucket:    aws.String(bucket),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	}

	for {
		out, err := c.client.ListObjectsV2(context.Background(), input)
		if err != nil {
			return nil, fmt.Errorf("list objects (prefix=%s): %w", prefix, err)
		}

		for _, cp := range out.CommonPrefixes {
			if cp.Prefix == nil {
				continue
			}

			fmt.Printf("Common prefix: %s\n", *cp.Prefix)
			name := strings.TrimSuffix(strings.TrimPrefix(*cp.Prefix, prefix), "/")
			if name == "" {
				continue
			}
			if _, ok := seen[name]; !ok {
				seen[name] = struct{}{}
				children = append(children, name)
			}
		}

		if out.IsTruncated != nil && !*out.IsTruncated {
			break
		}
		input.ContinuationToken = out.NextContinuationToken
	}

	fmt.Printf("Children: %v\n", children)
	return children, nil
}

// UploadDirectory uploads all files from a local directory to S3.
func (c *S3Client) UploadDirectory(bucket, prefix string, localPath string, fs fs.Filesystem) error {
	if c.walker == nil {
		return fmt.Errorf("walker not initialized")
	}

	c.logger.Info("Uploading directory to S3", "bucket", bucket, "prefix", prefix, "localPath", localPath)

	return c.walker.Walk(localPath, func(path string, fileType walker.FileType, getReader func() (walker.FileSeeker, error)) error {
		if fileType != walker.FileTypeFile {
			return nil
		}

		relPath, err := filepath.Rel(localPath, path)
		if err != nil {
			return fmt.Errorf("failed to calculate relative path: %w", err)
		}

		s3Key := filepath.Join(prefix, relPath)
		s3Key = filepath.ToSlash(s3Key)

		reader, err := getReader()
		if err != nil {
			return fmt.Errorf("failed to get file reader for %s: %w", path, err)
		}
		defer reader.Close()

		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		c.logger.Debug("Uploading file", "localPath", path, "s3Key", s3Key, "contentType", contentType)
		if err := c.UploadFile(bucket, s3Key, reader, contentType); err != nil {
			return fmt.Errorf("failed to upload file %s: %w", path, err)
		}

		return nil
	})
}

// NewS3Client returns a new S3 client.
func NewS3Client(logger *slog.Logger) (S3Client, error) {
	cfg, err := NewConfig()
	if err != nil {
		return S3Client{}, err
	}

	c := s3.NewFromConfig(cfg)
	w := walker.NewDefaultFSWalker(logger)

	return S3Client{
		client: c,
		logger: logger,
		walker: &w,
	}, nil
}

// NewCustomS3Client returns a new custom S3 client.
func NewCustomS3Client(client AWSS3Client, walker walker.Walker, logger *slog.Logger) S3Client {
	return S3Client{
		client: client,
		logger: logger,
		walker: walker,
	}
}
