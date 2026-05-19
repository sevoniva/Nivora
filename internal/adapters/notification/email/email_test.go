package email

import (
	"context"
	"testing"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

func TestProviderPlaceholderRequiresConfiguration(t *testing.T) {
	err := New(false).Send(context.Background(), domainnotification.Notification{Subject: "test"})
	if err == nil {
		t.Fatal("expected unconfigured email provider to reject send")
	}
}

func TestProviderConfiguredNoopsSafely(t *testing.T) {
	if err := New(true).Send(context.Background(), domainnotification.Notification{Subject: "test"}); err != nil {
		t.Fatalf("send: %v", err)
	}
}
