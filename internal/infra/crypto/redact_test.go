package crypto

import "testing"

func TestRedactMapRedactsSensitiveKeys(t *testing.T) {
	values := map[string]string{
		"username":      "example-user",
		"password":      "placeholder",
		"authorization": "Bearer placeholder",
	}
	redactedValues := RedactMap(values)
	if redactedValues["username"] != "example-user" {
		t.Fatalf("expected non-sensitive value to remain")
	}
	if redactedValues["password"] != redacted || redactedValues["authorization"] != redacted {
		t.Fatalf("expected sensitive values to be redacted: %#v", redactedValues)
	}
}

func TestRedactStringRedactsBearerText(t *testing.T) {
	if got := RedactString("bearer placeholder"); got != redacted {
		t.Fatalf("expected bearer text to be redacted, got %q", got)
	}
}

func TestRedactMapRedactsOperationalCredentialMarkers(t *testing.T) {
	values := map[string]string{
		"trace_id":    "trace-1",
		"kubeconfig":  "apiVersion: v1",
		"access_key":  "placeholder-access-key",
		"private_key": "placeholder-private-key",
	}
	redactedValues := RedactMap(values)
	if redactedValues["trace_id"] != "trace-1" {
		t.Fatalf("expected trace id to remain")
	}
	for _, key := range []string{"kubeconfig", "access_key", "private_key"} {
		if redactedValues[key] != redacted {
			t.Fatalf("expected %s to be redacted: %#v", key, redactedValues)
		}
	}
}
