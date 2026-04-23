package trips

import (
	"context"
	"strings"
	"testing"
)

func TestRedactAppSettingsForExport(t *testing.T) {
	in := AppSettings{
		AppTitle:         "Test",
		GoogleMapsAPIKey: "secret-key-123",
		AirLabsAPIKey:    "airlabs-secret-key",
	}
	out := RedactAppSettingsForExport(in)
	if out.GoogleMapsAPIKey != "[REDACTED]" {
		t.Fatalf("expected redacted maps key, got %q", out.GoogleMapsAPIKey)
	}
	if out.AirLabsAPIKey != "[REDACTED]" {
		t.Fatalf("expected redacted airlabs key, got %q", out.AirLabsAPIKey)
	}
	if in.GoogleMapsAPIKey != "secret-key-123" {
		t.Fatalf("caller value should be unchanged (pass-by-value), got %q", in.GoogleMapsAPIKey)
	}
	empty := RedactAppSettingsForExport(AppSettings{})
	if empty.GoogleMapsAPIKey != "" {
		t.Fatalf("empty key should stay empty, got %q", empty.GoogleMapsAPIKey)
	}
}

func TestBuildAccountExportEmptyUser(t *testing.T) {
	s := &Service{}
	_, err := s.BuildAccountExport(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty user id")
	}
	if !strings.Contains(err.Error(), "user id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildAccountExportWhitespaceUser(t *testing.T) {
	s := &Service{}
	_, err := s.BuildAccountExport(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected error for blank user id")
	}
}
