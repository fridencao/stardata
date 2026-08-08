// Package assetstore abstracts object storage for user-uploaded assets (project archives, images).
// It has two implementations: GCS (Rill Cloud 原生方案) and Local (私有化部署的本地磁盘方案)。
// 两者都通过"签名 URL"语义对外提供直传/直下能力，CLI 与 runtime 无需感知具体实现。
package assetstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
)

// Store abstracts asset object storage.
// objectPath is the bucket-relative path of the object (no leading slash).
type Store interface {
	// ObjectURL returns the canonical asset path persisted in the database,
	// e.g. "gs://<bucket>/<objectPath>" or "file://local/<objectPath>".
	ObjectURL(objectPath string) string
	// GenerateUploadURL returns a pre-signed PUT URL and the headers the uploader must send.
	GenerateUploadURL(objectPath string, maxSize int64) (signedURL string, headers map[string]string, err error)
	// GenerateDownloadURL returns a pre-signed GET URL valid for the given duration.
	GenerateDownloadURL(objectPath string, expires time.Duration) (string, error)
	// NewReader opens the object for reading.
	NewReader(ctx context.Context, objectPath string) (io.ReadCloser, error)
	// Delete removes the object. It must not return an error if the object doesn't exist.
	Delete(ctx context.Context, objectPath string) error
}

// GCS implements Store backed by a Google Cloud Storage bucket.
type GCS struct {
	bucket *storage.BucketHandle
}

// NewGCS creates a GCS-backed asset store.
func NewGCS(bucket *storage.BucketHandle) *GCS {
	return &GCS{bucket: bucket}
}

// ObjectURL implements Store.
func (g *GCS) ObjectURL(objectPath string) string {
	u := &url.URL{
		Scheme: "gs",
		Host:   g.bucket.BucketName(),
		Path:   objectPath,
	}
	return u.String()
}

// GenerateUploadURL implements Store.
// The returned headers enforce a maximum asset size for uploads.
func (g *GCS) GenerateUploadURL(objectPath string, maxSize int64) (string, map[string]string, error) {
	headers := map[string]string{
		"Content-Type":                "application/octet-stream",
		"x-goog-content-length-range": fmt.Sprintf("1,%d", maxSize),
	}
	var signingHeaders []string
	for k, v := range headers {
		signingHeaders = append(signingHeaders, fmt.Sprintf("%s:%s", k, v))
	}
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "PUT",
		Headers: signingHeaders,
		Expires: time.Now().Add(15 * time.Minute),
	}
	signedURL, err := g.bucket.SignedURL(objectPath, opts)
	if err != nil {
		return "", nil, err
	}
	return signedURL, headers, nil
}

// GenerateDownloadURL implements Store.
func (g *GCS) GenerateDownloadURL(objectPath string, expires time.Duration) (string, error) {
	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expires),
	}
	return g.bucket.SignedURL(objectPath, opts)
}

// NewReader implements Store.
func (g *GCS) NewReader(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	return g.bucket.Object(objectPath).NewReader(ctx)
}

// Delete implements Store.
func (g *GCS) Delete(ctx context.Context, objectPath string) error {
	err := g.bucket.Object(objectPath).Delete(ctx)
	if err != nil && !errors.Is(err, storage.ErrObjectNotExist) {
		return err
	}
	return nil
}
