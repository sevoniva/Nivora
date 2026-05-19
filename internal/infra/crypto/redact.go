package crypto

import "strings"

const redacted = "[REDACTED]"

var sensitiveMarkers = []string{
	"token",
	"password",
	"secret",
	"private_key",
	"privatekey",
	"kubeconfig",
	"authorization",
	"access_key",
	"accesskey",
	"secret_key",
	"secretkey",
	"client_secret",
	"refresh_token",
	"id_token",
	"session",
	"bearer",
}

func RedactString(value string) string {
	lower := strings.ToLower(value)
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return redacted
		}
	}
	return value
}

func RedactMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		if IsSensitiveKey(key) {
			out[key] = redacted
			continue
		}
		out[key] = RedactString(value)
	}
	return out
}

func IsSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, marker := range sensitiveMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
