// Package s3 provides an S3-compatible object store adapter (AWS S3, MinIO, etc.).
// Uses standard HTTP calls with AWS Signature V4. No AWS SDK dependency.
package s3

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sevoniva/nivora/internal/ports/objectstore"
)

var (
	ErrNotFound  = errors.New("object not found")
	ErrForbidden = errors.New("access denied")
)

type Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Insecure  bool
}

type Store struct {
	cfg    Config
	client *http.Client
}

var _ objectstore.ObjectStore = (*Store)(nil)

func New(cfg Config) *Store {
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	return &Store{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewMinIO returns a store configured for local MinIO (defaults).
// Endpoint can be "localhost:9000" or "http://localhost:9000".
func NewMinIO(endpoint, accessKey, secretKey, bucket string) *Store {
	insecure := !strings.HasPrefix(endpoint, "https://")
	// Strip scheme for endpoint storage.
	clean := endpoint
	clean = strings.TrimPrefix(clean, "http://")
	clean = strings.TrimPrefix(clean, "https://")
	return New(Config{
		Endpoint:  clean,
		Region:    "us-east-1",
		Bucket:    bucket,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Insecure:  insecure,
	})
}

func (s *Store) objectURL(key string) string {
	scheme := "https"
	if s.cfg.Insecure {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, s.cfg.Endpoint, s.cfg.Bucket, key)
}

func (s *Store) PutObject(ctx context.Context, key string, body io.Reader) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, s.objectURL(key), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.ContentLength = int64(len(data))

	if err := s.signRequest(req, strings.NewReader(string(data))); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	return s.handleError(resp)
}

func (s *Store) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.objectURL(key), nil)
	if err != nil {
		return nil, err
	}

	if err := s.signRequest(req, nil); err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}

	if resp.StatusCode == http.StatusOK {
		return resp.Body, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	return nil, s.handleError(resp)
}

func (s *Store) DeleteObject(ctx context.Context, key string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, s.objectURL(key), nil)
	if err != nil {
		return err
	}

	if err := s.signRequest(req, nil); err != nil {
		return err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return s.handleError(resp)
}

func (s *Store) handleError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = fmt.Sprintf("S3 error: HTTP %d", resp.StatusCode)
	}
	switch resp.StatusCode {
	case http.StatusNotFound:
		return fmt.Errorf("%s: %w", msg, ErrNotFound)
	case http.StatusForbidden:
		return fmt.Errorf("%s: %w", msg, ErrForbidden)
	default:
		return errors.New(msg)
	}
}

// --- AWS Signature V4 ---

func (s *Store) signRequest(req *http.Request, body io.Reader) error {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")

	hashedPayload := "UNSIGNED-PAYLOAD"
	if body != nil {
		h := sha256.New()
		io.Copy(h, body)
		hashedPayload = hex.EncodeToString(h.Sum(nil))
	}
	req.Header.Set("x-amz-content-sha256", hashedPayload)
	req.Header.Set("x-amz-date", amzDate)
	if req.Header.Get("Host") == "" {
		req.Header.Set("Host", req.URL.Host)
	}

	canonicalRequest := s.canonicalRequest(req, hashedPayload)
	stringToSign := s.stringToSign(now, canonicalRequest)
	signature := s.calculateSignature(now, stringToSign)

	credential := fmt.Sprintf("%s/%s/%s/s3/aws4_request", s.cfg.AccessKey, dateStamp, s.cfg.Region)
	auth := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s, SignedHeaders=%s, Signature=%s",
		credential, s.signedHeaders(req), signature)
	req.Header.Set("Authorization", auth)
	return nil
}

func (s *Store) canonicalRequest(req *http.Request, payloadHash string) string {
	var b strings.Builder
	b.WriteString(req.Method + "\n")
	fmt.Fprintf(&b, "%s\n", req.URL.EscapedPath())
	if req.URL.RawQuery != "" {
		fmt.Fprintf(&b, "%s\n", req.URL.RawQuery)
	} else {
		b.WriteString("\n")
	}

	signedHeaders := s.signedHeaders(req)
	for _, h := range strings.Split(signedHeaders, ";") {
		fmt.Fprintf(&b, "%s:%s\n", h, strings.TrimSpace(req.Header.Get(h)))
	}
	b.WriteString("\n")
	b.WriteString(signedHeaders + "\n")
	b.WriteString(payloadHash)
	return b.String()
}

func (s *Store) signedHeaders(req *http.Request) string {
	return "host;x-amz-content-sha256;x-amz-date"
}

func (s *Store) stringToSign(now time.Time, canonicalRequest string) string {
	hash := sha256.Sum256([]byte(canonicalRequest))
	return fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s/%s/s3/aws4_request\n%s",
		now.Format("20060102T150405Z"),
		now.Format("20060102"), s.cfg.Region,
		hex.EncodeToString(hash[:]))
}

func (s *Store) calculateSignature(now time.Time, stringToSign string) string {
	dateKey := hmacSHA256([]byte("AWS4"+s.cfg.SecretKey), now.Format("20060102"))
	regionKey := hmacSHA256(dateKey, s.cfg.Region)
	serviceKey := hmacSHA256(regionKey, "s3")
	signingKey := hmacSHA256(serviceKey, "aws4_request")
	return hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
}

func hmacSHA256(key []byte, data string) []byte {
	h := hmac.New(sha256.New, key)
	h.Write([]byte(data))
	return h.Sum(nil)
}
