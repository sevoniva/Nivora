package yamlapply

import (
	"context"
	"testing"

	deploymentusecase "github.com/sevoniva/nivora/internal/usecase/deployment"
)

func TestNoopManifestClientDryRunApplyAndRollout(t *testing.T) {
	client := NoopManifestClient{}
	request := deploymentusecase.ManifestRequest{
		Plan: deploymentusecase.DeploymentPlan{
			DeploymentRunID: "drun-test",
			Resources: []deploymentusecase.ManifestResourceSummary{{
				APIVersion: "v1",
				Kind:       "ConfigMap",
				Name:       "demo",
			}},
		},
		Documents: []deploymentusecase.ManifestDocument{{Content: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: demo\n"}},
	}
	if result, err := client.ServerDryRun(context.Background(), request); err != nil || result.Message == "" {
		t.Fatalf("server dry-run result=%#v err=%v", result, err)
	}
	if result, err := client.Apply(context.Background(), request); err != nil || result.Message == "" {
		t.Fatalf("apply result=%#v err=%v", result, err)
	}
	if result, err := client.WatchRollout(context.Background(), request); err != nil || result.Message == "" {
		t.Fatalf("rollout result=%#v err=%v", result, err)
	}
	if result, err := client.Rollback(context.Background(), request); err != nil || result.Message == "" {
		t.Fatalf("rollback result=%#v err=%v", result, err)
	}
}

func TestNoopManifestClientRejectsEmptyRequest(t *testing.T) {
	if _, err := (NoopManifestClient{}).ServerDryRun(context.Background(), deploymentusecase.ManifestRequest{}); err == nil {
		t.Fatal("expected validation error")
	}
}
