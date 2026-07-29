package assetstore

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Local 实现基于本地磁盘的 asset 存储（私有化部署，无对象存储依赖）。
// "签名 URL" 指向 admin 自身的 HTTP 端点，用 HMAC-SHA256 令牌自鉴权，
// 与 GCS 签名 URL 语义一致：CLI 裸 PUT 上传、runtime 裸 GET 下载均无需改动。
type Local struct {
	dir         string
	externalURL string
	secret      []byte
}

// HTTP paths served by the admin server for the local asset store.
const (
	LocalUploadPath   = "/v1/assets/-/upload"
	LocalDownloadPath = "/v1/assets/-/download"
)

// NewLocal creates a Local asset store rooted at dir.
// externalURL is the admin server's external URL (used as the base of signed URLs).
// secret is the HMAC signing key for the signed URLs.
func NewLocal(dir, externalURL string, secret []byte) (*Local, error) {
	if dir == "" {
		return nil, errors.New("assetstore: dir is required")
	}
	if externalURL == "" {
		return nil, errors.New("assetstore: external URL is required")
	}
	if len(secret) == 0 {
		return nil, errors.New("assetstore: signing secret is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("assetstore: failed to create dir %q: %w", dir, err)
	}
	return &Local{
		dir:         dir,
		externalURL: strings.TrimSuffix(externalURL, "/"),
		secret:      secret,
	}, nil
}

// ObjectURL implements Store.
// The "local" host is a fixed placeholder so the path parses like "gs://<bucket>/<path>".
func (l *Local) ObjectURL(objectPath string) string {
	return "file://local/" + objectPath
}

// GenerateUploadURL implements Store.
func (l *Local) GenerateUploadURL(objectPath string, maxSize int64) (string, map[string]string, error) {
	if _, err := l.safePath(objectPath); err != nil {
		return "", nil, err
	}
	exp := time.Now().Add(15 * time.Minute).Unix()
	q := url.Values{}
	q.Set("path", objectPath)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("max", strconv.FormatInt(maxSize, 10))
	q.Set("sig", l.sign(http.MethodPut, objectPath, exp, maxSize))
	headers := map[string]string{"Content-Type": "application/octet-stream"}
	return l.externalURL + LocalUploadPath + "?" + q.Encode(), headers, nil
}

// GenerateDownloadURL implements Store.
func (l *Local) GenerateDownloadURL(objectPath string, expires time.Duration) (string, error) {
	if _, err := l.safePath(objectPath); err != nil {
		return "", err
	}
	exp := time.Now().Add(expires).Unix()
	q := url.Values{}
	q.Set("path", objectPath)
	q.Set("exp", strconv.FormatInt(exp, 10))
	q.Set("sig", l.sign(http.MethodGet, objectPath, exp, 0))
	return l.externalURL + LocalDownloadPath + "?" + q.Encode(), nil
}

// VerifyURL validates the query params of a signed local asset URL.
// It returns the object path and (for uploads) the max allowed size.
func (l *Local) VerifyURL(method string, q url.Values) (objectPath string, maxSize int64, err error) {
	objectPath = q.Get("path")
	exp, err := strconv.ParseInt(q.Get("exp"), 10, 64)
	if err != nil {
		return "", 0, errors.New("assetstore: invalid expiry")
	}
	if method == http.MethodPut {
		maxSize, err = strconv.ParseInt(q.Get("max"), 10, 64)
		if err != nil || maxSize <= 0 {
			return "", 0, errors.New("assetstore: invalid max size")
		}
	}
	if time.Now().Unix() > exp {
		return "", 0, errors.New("assetstore: signed URL has expired")
	}
	expected := l.sign(method, objectPath, exp, maxSize)
	if !hmac.Equal([]byte(expected), []byte(q.Get("sig"))) {
		return "", 0, errors.New("assetstore: invalid signature")
	}
	if _, err := l.safePath(objectPath); err != nil {
		return "", 0, err
	}
	return objectPath, maxSize, nil
}

// Write atomically persists the object (temp file + rename).
func (l *Local) Write(objectPath string, data io.Reader) error {
	dst, err := l.safePath(objectPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".upload-*")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, data)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// NewReader implements Store.
func (l *Local) NewReader(ctx context.Context, objectPath string) (io.ReadCloser, error) {
	p, err := l.safePath(objectPath)
	if err != nil {
		return nil, err
	}
	return os.Open(p)
}

// Delete implements Store.
func (l *Local) Delete(ctx context.Context, objectPath string) error {
	p, err := l.safePath(objectPath)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// sign computes the HMAC-SHA256 signature for a signed URL.
func (l *Local) sign(method, objectPath string, exp, maxSize int64) string {
	mac := hmac.New(sha256.New, l.secret)
	fmt.Fprintf(mac, "%s\n%s\n%d\n%d", method, objectPath, exp, maxSize)
	return hex.EncodeToString(mac.Sum(nil))
}

// safePath resolves objectPath under the store dir and rejects path traversal.
func (l *Local) safePath(objectPath string) (string, error) {
	if objectPath == "" || path.IsAbs(objectPath) {
		return "", errors.New("assetstore: invalid object path")
	}
	clean := path.Clean(objectPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("assetstore: invalid object path")
	}
	// Defense in depth: verify the final resolved path stays inside the store dir.
	resolved := filepath.Join(l.dir, filepath.FromSlash(clean))
	rel, err := filepath.Rel(l.dir, resolved)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("assetstore: invalid object path")
	}
	return resolved, nil
}
