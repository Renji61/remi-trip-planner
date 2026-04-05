package httpapp

import (
	"net/http"
	"strings"
	"testing"
)

func TestTruncatedUserAgent(t *testing.T) {
	long := strings.Repeat("a", maxUserAgentLogLen+50)
	r := &http.Request{Header: http.Header{}}
	r.Header.Set("User-Agent", long)
	got := TruncatedUserAgent(r)
	if len(got) != maxUserAgentLogLen {
		t.Fatalf("len %d, want %d", len(got), maxUserAgentLogLen)
	}
	if TruncatedUserAgent(nil) != "" {
		t.Fatal("nil request")
	}
}
