package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func TestRequestContextAddsCorrelationAndTrace(t *testing.T) {
	handler := chimiddleware.RequestID(RequestContext()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if CorrelationID(r.Context()) != "corr-1" {
			t.Fatalf("correlation id = %q", CorrelationID(r.Context()))
		}
		if TraceID(r.Context()) != "0123456789abcdef0123456789abcdef" {
			t.Fatalf("trace id = %q", TraceID(r.Context()))
		}
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderCorrelationID, "corr-1")
	req.Header.Set("traceparent", "00-0123456789abcdef0123456789abcdef-0123456789abcdef-01")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get(HeaderCorrelationID) != "corr-1" {
		t.Fatalf("correlation header = %q", rec.Header().Get(HeaderCorrelationID))
	}
	if rec.Header().Get(HeaderTraceID) != "0123456789abcdef0123456789abcdef" {
		t.Fatalf("trace header = %q", rec.Header().Get(HeaderTraceID))
	}
}
