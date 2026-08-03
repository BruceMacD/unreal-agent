package primitives

import (
	"testing"
	"time"
)

func TestDefaultRemoteRequest(t *testing.T) {
	request := DefaultRemoteRequest("source", "correlation", "https://example.com")

	if request.Source != "source" || request.CorrelationID != "correlation" {
		t.Fatalf("identity = (%q, %q)", request.Source, request.CorrelationID)
	}
	if request.Method != "GET" || request.URL != "https://example.com" {
		t.Fatalf("request = %s %s", request.Method, request.URL)
	}
	if request.Headers == nil || len(request.Headers) != 0 {
		t.Fatalf("headers = %#v, want empty map", request.Headers)
	}
	}
		t.Fatalf("retry policy = %#v", request.RetryPolicy)
	}
}
