package s3

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sevoniva/nivora/internal/ports/objectstore"
)

func TestS3StoreImplementsInterface(t *testing.T) {
	var _ objectstore.ObjectStore = New(Config{
		Endpoint: "localhost:9000", Region: "us-east-1", Bucket: "test",
		AccessKey: "minioadmin", SecretKey: "minioadmin", Insecure: true,
	})
}

func TestNewMinIO(t *testing.T) {
	s := NewMinIO("localhost:9000", "minioadmin", "minioadmin", "nivora-test")
	if s == nil {
		t.Fatal("NewMinIO returned nil")
	}
	if !s.cfg.Insecure {
		t.Fatal("expected insecure for http:// endpoint")
	}
}

func TestNewMinIOHTTPS(t *testing.T) {
	s := NewMinIO("https://s3.amazonaws.com", "AKID", "SECRET", "bucket")
	if s.cfg.Insecure {
		t.Fatal("expected secure for https:// endpoint")
	}
}

func TestObjectURLFormat(t *testing.T) {
	s := NewMinIO("http://localhost:9000", "k", "s", "my-bucket")
	u := s.objectURL("path/to/object.txt")
	expected := "http://localhost:9000/my-bucket/path/to/object.txt"
	if u != expected {
		t.Fatalf("expected %s, got %s", expected, u)
	}
}

func TestErrorNotFound(t *testing.T) {
	if !strings.Contains(ErrNotFound.Error(), "not found") {
		t.Fatal("ErrNotFound should mention 'not found'")
	}
}

func TestErrorForbidden(t *testing.T) {
	if !strings.Contains(ErrForbidden.Error(), "denied") {
		t.Fatal("ErrForbidden should mention 'denied'")
	}
}

// --- Integration test (requires MinIO) ---
func TestS3PutGetDeleteWithMinIO(t *testing.T) {
	endpoint := os.Getenv("MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("MINIO_ENDPOINT not set")
	}
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}
	bucket := os.Getenv("MINIO_BUCKET")
	if bucket == "" {
		bucket = "nivora-test"
	}

	s := NewMinIO(endpoint, accessKey, secretKey, bucket)
	ctx := context.Background()

	// Put
	key := "nivora-test/s3-adapter-test.txt"
	data := "S3 adapter integration test"
	err := s.PutObject(ctx, key, strings.NewReader(data))
	if err != nil {
		t.Fatalf("put object: %v", err)
	}

	// Get
	reader, err := s.GetObject(ctx, key)
	if err != nil {
		t.Fatalf("get object: %v", err)
	}
	defer reader.Close()
	got, _ := io.ReadAll(reader)
	if string(got) != data {
		t.Fatalf("expected %q, got %q", data, string(got))
	}

	// Delete
	if err := s.DeleteObject(ctx, key); err != nil {
		t.Fatalf("delete object: %v", err)
	}

	// Verify deleted
	_, err = s.GetObject(ctx, key)
	if err == nil {
		t.Fatal("expected error after delete")
	}
	t.Logf("delete confirmed: %v", err)
}
