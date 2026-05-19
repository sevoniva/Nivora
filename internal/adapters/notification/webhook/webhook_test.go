package webhook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

func TestProviderRequiresExplicitAllow(t *testing.T) {
	provider := New("http://example.invalid/webhook", false)
	err := provider.Send(context.Background(), domainnotification.Notification{Subject: "test"})
	if err == nil {
		t.Fatal("expected disabled webhook provider to reject send")
	}
}

func TestProviderSendsWhenAllowed(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s", r.Method)
		}
		if r.Header.Get("content-type") != "application/json" {
			t.Fatalf("content-type = %s", r.Header.Get("content-type"))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	provider := New(server.URL, true)
	if err := provider.Send(context.Background(), domainnotification.Notification{Subject: "test"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !called {
		t.Fatal("expected webhook server to be called")
	}
}
