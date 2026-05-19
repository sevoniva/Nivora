package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	domainnotification "github.com/sevoniva/nivora/internal/domain/notification"
)

type Provider struct {
	Endpoint  string
	AllowSend bool
	Client    *http.Client
}

func New(endpoint string, allowSend bool) Provider {
	return Provider{Endpoint: endpoint, AllowSend: allowSend}
}

func (p Provider) Send(ctx context.Context, notification domainnotification.Notification) error {
	if !p.AllowSend {
		return errors.New("webhook notification is disabled; enable it explicitly before sending")
	}
	if p.Endpoint == "" {
		return errors.New("webhook endpoint is required")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.Endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook notification failed with status %d", resp.StatusCode)
	}
	return nil
}
