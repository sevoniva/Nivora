package kms

import (
	"context"
	"testing"
)

func TestKMSPlaceholdersValidateWithoutCloud(t *testing.T) {
	for _, provider := range []*Provider{NewAWS(), NewAliyun(), NewTencent()} {
		status, err := provider.ValidateProvider(context.Background())
		if err != nil {
			t.Fatalf("validate provider: %v", err)
		}
		if status.Configured || status.Reachable {
			t.Fatalf("placeholder should not be configured/reachable: %#v", status)
		}
	}
}
