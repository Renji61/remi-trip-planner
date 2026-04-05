package httpapp

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"remi-trip-planner/internal/trips"
)

func TestTripDocumentsRequestWantsJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/trips/t1/documents/upload", nil)
	if tripDocumentsRequestWantsJSON(r) {
		t.Fatal("expected false without headers")
	}
	r.Header.Set("X-Requested-With", "XMLHttpRequest")
	if tripDocumentsRequestWantsJSON(r) {
		t.Fatal("expected false without Accept json")
	}
	r.Header.Set("Accept", "application/json")
	if !tripDocumentsRequestWantsJSON(r) {
		t.Fatal("expected true with XHR + Accept application/json")
	}
}

func TestTripDocumentRowFromTripDocument(t *testing.T) {
	d := trips.TripDocument{
		ID:          "doc-1",
		Section:     "general",
		Category:    "General Documents",
		ItemName:    "Paris",
		FileName:    "ticket.pdf",
		DisplayName: "Boarding pass",
		FilePath:    "/uploads/trips/x/ticket.pdf",
		FileSize:    1024,
		UploadedAt:  time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC),
	}
	row := tripDocumentRowFromTripDocument(d)
	if row.ID != d.ID || row.Section != "general" {
		t.Fatalf("row: %+v", row)
	}
	if !strings.Contains(row.SearchText, "boarding") || !strings.Contains(row.SearchText, "paris") {
		t.Fatalf("search text should include keywords, got %q", row.SearchText)
	}
	if row.FileKind != "pdf" || row.FileTypeIcon != "picture_as_pdf" {
		t.Fatalf("file visual: kind=%q icon=%q", row.FileKind, row.FileTypeIcon)
	}
}

func TestTripDocumentRowFromTripDocumentEmptyPath(t *testing.T) {
	row := tripDocumentRowFromTripDocument(trips.TripDocument{ID: "x", FilePath: "  "})
	if row.ID != "" {
		t.Fatalf("expected empty row for empty path, got %+v", row)
	}
}
